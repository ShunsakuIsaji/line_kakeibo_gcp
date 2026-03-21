package gemini

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type geminiRequest struct {
	contents []struct {
		parts []struct {
			inlineData struct {
				mimeType string `json:"mimeType"`
				data     string `json:"data"`
			} `json:"inlineData"`
			text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	generetionConfig struct {
		responseMimeType string `json:"responseMimeType"`
		responseSchema   struct {
			typeof     string `json:"type"`
			properties struct {
				Date struct {
					typeof      string `json:"type"`
					description string `json:"description"`
				} `json:"Date"`
				TotalAmount struct {
					typeof      string `json:"type"`
					description string `json:"description"`
				} `json:"TotalAmount"`
				Shopname struct {
					typeof      string `json:"type"`
					description string `json:"description"`
				} `json:"Shopname"`
				Category struct {
					typeof      string `json:"type"`
					description string `json:"description"`
				} `json:"Category"`
				Memo struct {
					typeof      string `json:"type"`
					description string `json:"description"`
				} `json:"Memo"`
			} `json:"properties"`
			required []string `json:"required"`
		} `json:"responseSchema"`
	} `json:"generationConfig"`
}

type geminiResponse struct {
	Date        string `json:"Date"`
	TotalAmount string `json:"TotalAmount"`
	Shopname    string `json:"Shopname"`
	Category    string `json:"Category"`
	Memo        string `json:"Memo"`
}

func getGeminiRequestBody(imageData string) *geminiRequest {
	requestBody := new(geminiRequest)
	requestBody.contents[0].parts[0].inlineData.mimeType = "image/jpeg"
	requestBody.contents[0].parts[0].inlineData.data = imageData
	requestBody.contents[0].parts[0].text = `
	あなたはレシートの画像から情報を抽出するプロフェッショナルです。
添付された画像を分析し、分析結果をAPI設定で指定されたJSONスキーマで出力してください。
- "Date": レシートに記載されている購入日付 (YYYY-MM-DD形式)。記載されていない場合、読み取れない場合は空文字列("")で構いません。
- "TotalAmount": レシートに記載されている税込の合計金額 (数値)。
- "ShopName": レシートに記載されている店舗名（スーパー名、店名など）。
- "Category": 購入品目のカテゴリを「食費・外食・日用品・その他」のうち最も近いもの。 食費はスーパーや食材店などの食料品購入、外食はレストランやカフェなどでの飲食、日用品は生活必需品の購入、その他は上記以外のカテゴリを指す。
- "Memo": レシートにトイレットペーパー、ティッシュペーパー、衣類用洗剤、衣類用柔軟剤、食器用洗剤が含まれている場合、それぞれ「トイレットペーパー購入」、「ティッシュ購入」、「衣類用洗剤購入」、「柔軟剤購入」、「食器用洗剤購入」と出力し、含まれていない場合は「なし」と出力。複数当てはまる場合はカンマ区切りで全て出力する。
 
分析の際は、特に "TotalAmount" と "ShopName" の抽出精度を最大限に高めてください。
`
	requestBody.generetionConfig.responseMimeType = "application/json"
	requestBody.generetionConfig.responseSchema.typeof = "object"
	requestBody.generetionConfig.responseSchema.properties.Date.typeof = "string"
	requestBody.generetionConfig.responseSchema.properties.Date.description = "画像のレシートから読み取れる購入日付をYYYY-MM-DD形式で出力する"
	requestBody.generetionConfig.responseSchema.properties.TotalAmount.typeof = "number"
	requestBody.generetionConfig.responseSchema.properties.TotalAmount.description = "画像のレシートから読み取れる税込の合計金額を数値で出力する"
	requestBody.generetionConfig.responseSchema.properties.Shopname.typeof = "string"
	requestBody.generetionConfig.responseSchema.properties.Shopname.description = "画像のレシートから読み取れる店舗名（スーパー名、店名など）を出力する"
	requestBody.generetionConfig.responseSchema.properties.Category.typeof = "string"
	requestBody.generetionConfig.responseSchema.properties.Category.description = "画像のレシートから読み取れる購入品目のカテゴリを、「食費・外食・日用品・その他」のうち最も近いものを出力する。食費はスーパーや食材店などの食料品購入、外食はレストランやカフェなどでの飲食、日用品は生活必需品の購入、その他は上記以外のカテゴリを指す"
	requestBody.generetionConfig.responseSchema.properties.Memo.typeof = "string"
	requestBody.generetionConfig.responseSchema.properties.Memo.description = "レシートから、トイレットペーパーが含まれていれば「トイレットペーパー購入」、ティッシュペーパーが含まれていれば「ティッシュ購入」、衣類用洗剤が含まれていれば「衣類用洗剤購入」、衣類用柔軟剤が含まれていれば「柔軟剤購入」、食器用洗剤が含まれていれば「食器用洗剤購入」、特に含まれていなければ「なし」と出力する。複数当てはまる場合は、カンマ区切りで全て出力する"
	requestBody.generetionConfig.responseSchema.required = []string{"Date", "TotalAmount", "Shopname", "Category", "Memo"}
	return requestBody
}

func sendRequestToGemini(endpoint string, body []byte) ([]byte, error) {

	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("failed to send request to Gemini : %S", err)
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response from Gemini : %S", err)
		return nil, err
	}

	return responseBody, nil
}

func getGeminiResponse(GeminiEndpoint string, requestbody *geminiRequest) (*geminiResponse, error) {
	body, err := json.Marshal(requestbody)
	if err != nil {
		return nil, err
	}

	responseBody, err := sendRequestToGemini(GeminiEndpoint, body)
	if err != nil {
		return nil, err
	}

	var geminiResponse geminiResponse
	err = json.Unmarshal(responseBody, &geminiResponse)
	if err != nil {
		return nil, err
	}

	return &geminiResponse, nil
}
