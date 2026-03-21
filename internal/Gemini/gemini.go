package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type GeminiPart struct {
	InlineData *GeminiInlineData `json:"inlineData,omitempty"`
	Text       *string           `json:"text,omitempty"`
}

type GeminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type GeminiContents struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPropaty struct {
	Typeof      string `json:"type"`
	Description string `json:"description"`
}

type GeminiPropaties struct {
	Date        GeminiPropaty `json:"date"`
	TotalAmount GeminiPropaty `json:"totalAmount"`
	ShopName    GeminiPropaty `json:"shopname"`
	Category    GeminiPropaty `json:"category"`
	Memo        GeminiPropaty `json:"memo"`
	Confidence  GeminiPropaty `json:"confidence,omitempty"`
}

type GeminiSchema struct {
	Typeof     string          `json:"type"`
	Properties GeminiPropaties `json:"properties"`
	Required   []string        `json:"required"`
}

type GeminiGenerationConfig struct {
	ResponseMimeType string       `json:"responseMimeType"`
	ResponseSchema   GeminiSchema `json:"responseSchema"`
}

type GeminiRequest struct {
	Contents         []GeminiContents       `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig"`
}

type GeminiResponse struct {
	Date        string   `json:"date"`
	TotalAmount int      `json:"totalAmount"`
	ShopName    string   `json:"shopname"`
	Category    string   `json:"category"`
	Memo        string   `json:"memo"`
	Confidence  *float64 `json:"confidence,omitempty"`
}

type GeminiAPIResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	// 他のフィールドは無視される
}

func GetGeminiRequestBody(imageData string) *GeminiRequest {
	prompt := `
	あなたはレシートの画像から情報を抽出するプロフェッショナルです。
添付された画像を分析し、分析結果をAPI設定で指定されたJSONスキーマで出力してください。
- "Date": レシートに記載されている購入日付 (YYYY-MM-DD形式)。記載されていない場合、読み取れない場合は空文字列("")で構いません。
- "TotalAmount": レシートに記載されている税込の合計金額 (数値)。
- "ShopName": レシートに記載されている店舗名（スーパー名、店名など）。
- "Category": 購入品目のカテゴリを「食費・外食・日用品・その他」のうち最も近いもの。 食費はスーパーや食材店などの食料品購入、外食はレストランやカフェなどでの飲食、日用品は生活必需品の購入、その他は上記以外のカテゴリを指す。
- "Memo": レシートにトイレットペーパー、ティッシュペーパー、衣類用洗剤、衣類用柔軟剤、食器用洗剤が含まれている場合、それぞれ「トイレットペーパー購入」、「ティッシュ購入」、「衣類用洗剤購入」、「柔軟剤購入」、「食器用洗剤購入」と出力し、含まれていない場合は「なし」と出力。複数当てはまる場合はカンマ区切りで全て出力する。
 
分析の際は、特に "TotalAmount" と "ShopName" の抽出精度を最大限に高めてください。
`
	return &GeminiRequest{
		Contents: []GeminiContents{
			{
				Parts: []GeminiPart{
					{
						InlineData: &GeminiInlineData{
							MimeType: "image/jpeg",
							Data:     imageData,
						},
					},
					{
						Text: &prompt,
					},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema: GeminiSchema{
				Typeof: "OBJECT",
				Properties: GeminiPropaties{
					Date: GeminiPropaty{
						Typeof:      "STRING",
						Description: "画像のレシートから読み取れる購入日付をYYYY-MM-DD形式で出力する",
					},
					TotalAmount: GeminiPropaty{
						Typeof:      "NUMBER",
						Description: "画像のレシートから読み取れる税込の合計金額を数値で出力する",
					},
					ShopName: GeminiPropaty{
						Typeof:      "STRING",
						Description: "画像のレシートから読み取れる店舗名（スーパー名、店名など）を出力する",
					},
					Category: GeminiPropaty{
						Typeof:      "STRING",
						Description: "画像のレシートから読み取れる購入品目のカテゴリを、「食費・外食・日用品・その他」のうち最も近いものを出力する。食費はスーパーや食材店などの食料品購入、外食はレストランやカフェなどでの飲食、日用品は生活必需品の購入、その他は上記以外のカテゴリを指す",
					},
					Memo: GeminiPropaty{
						Typeof:      "STRING",
						Description: "レシートから、トイレットペーパーが含まれていれば「トイレットペーパー購入」、ティッシュペーパーが含まれていれば「ティッシュ購入」、衣類用洗剤が含まれていれば「衣類用洗剤購入」、衣類用柔軟剤が含まれていれば「柔軟剤購入」、食器用洗剤が含まれていれば「食器用洗剤購入」、特に含まれていなければ「なし」と出力する。複数当てはまる場合は、カンマ区切りで全て出力する",
					},
					Confidence: GeminiPropaty{
						Typeof:      "NUMBER",
						Description: "Geminiが解析結果にどれだけ自信を持っているかを0から1の数値で出力する。1に近いほど自信が高いことを示す。不明な場合は省略しても構いません。",
					},
				},
				Required: []string{"date", "totalAmount", "shopname", "category", "memo"},
			},
		},
	}
}

func SendRequestToGemini(endpoint string, body []byte) ([]byte, error) {

	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("failed to send request to Gemini : %S", err)
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	log.Printf("gemini status=%d body=%s", resp.StatusCode, string(responseBody))

	if err != nil {
		log.Printf("failed to read response from Gemini : %S", err)
		return nil, err
	}

	return responseBody, nil
}

func GetGeminiResponse(GeminiEndpoint string, requestbody *GeminiRequest) (*GeminiResponse, error) {
	body, err := json.Marshal(requestbody)
	if err != nil {
		return nil, err
	}

	responseBody, err := SendRequestToGemini(GeminiEndpoint, body)
	if err != nil {
		return nil, err
	}

	var geminiResponse GeminiResponse
	var apiResponse GeminiAPIResponse
	err = json.Unmarshal(responseBody, &apiResponse)
	if err != nil {
		return nil, err
	}

	if len(apiResponse.Candidates) == 0 || len(apiResponse.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no candidates or parts in Gemini response")
	}

	// 最初の候補の最初のパートのテキストを解析結果とみなす
	geminiResult := apiResponse.Candidates[0].Content.Parts[0].Text
	// debag
	log.Printf("geminiResult raw=%q", geminiResult)

	if geminiResult == "" {
		return nil, fmt.Errorf("no text part in Gemini response")
	}
	err = json.Unmarshal([]byte(geminiResult), &geminiResponse)
	if err != nil {
		log.Printf("inner unmarshal error: %v", err)
		return nil, err
	}

	return &geminiResponse, nil
}
