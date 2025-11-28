package main

import (
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
)

// CreateClient initializes the client
func CreateClient() (*openapi.Client, error) {
	configMu.RLock()
	accessKeyID := appConfig.Aliyun.AccessKeyID
	accessKeySecret := appConfig.Aliyun.AccessKeySecret
	configMu.RUnlock()

	if accessKeyID == "" || accessKeySecret == "" {
		return nil, fmt.Errorf("Aliyun credentials not configured")
	}

	// Configure credentials
	credConfig := &credential.Config{
		Type:            tea.String("access_key"),
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
	}

	cred, err := credential.NewCredential(credConfig)
	if err != nil {
		return nil, err
	}

	config := &openapi.Config{
		Credential: cred,
	}
	config.Endpoint = tea.String("dypnsapi.aliyuncs.com")
	return openapi.NewClient(config)
}

// SendSmsCode sends a verification code to the phone number
func SendSmsCode(phone string) error {
	client, err := CreateClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	configMu.RLock()
	signName := appConfig.Aliyun.SMS.SignName
	templateCode := appConfig.Aliyun.SMS.TemplateCode
	configMu.RUnlock()

	if signName == "" || templateCode == "" {
		return fmt.Errorf("SMS configuration missing (sign_name or template_code)")
	}

	params := &openapi.Params{
		Action:      tea.String("SendSmsVerifyCode"),
		Version:     tea.String("2017-05-25"),
		Protocol:    tea.String("HTTPS"),
		Method:      tea.String("POST"),
		AuthType:    tea.String("AK"),
		Style:       tea.String("RPC"),
		Pathname:    tea.String("/"),
		ReqBodyType: tea.String("json"),
		BodyType:    tea.String("json"),
	}

	query := map[string]*string{
		"PhoneNumber":   tea.String(phone),
		"SignName":      tea.String(signName),
		"TemplateCode":  tea.String(templateCode),
		"TemplateParam": tea.String("{\"code\":\"##code##\",\"min\":\"5\"}"),
		"CodeLength":   tea.String("6"),
	}

	request := &openapi.OpenApiRequest{
		Query: query,
	}

	runtime := &util.RuntimeOptions{}

	resp, err := client.CallApi(params, request, runtime)
	if err != nil {
		return fmt.Errorf("CallApi failed: %v", err)
	}

	body, ok := resp["body"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response body type")
	}

	if code, ok := body["Code"].(string); ok && code != "OK" {
		msg, _ := body["Message"].(string)
		return fmt.Errorf("SMS API error: %s - %s", code, msg)
	}

	return nil
}

// CheckSmsCode verifies the code
func CheckSmsCode(phone, code string) (bool, error) {
	client, err := CreateClient()
	if err != nil {
		return false, fmt.Errorf("failed to create client: %v", err)
	}

	params := &openapi.Params{
		Action:      tea.String("CheckSmsVerifyCode"),
		Version:     tea.String("2017-05-25"),
		Protocol:    tea.String("HTTPS"),
		Method:      tea.String("POST"),
		AuthType:    tea.String("AK"),
		Style:       tea.String("RPC"),
		Pathname:    tea.String("/"),
		ReqBodyType: tea.String("json"),
		BodyType:    tea.String("json"),
	}

	query := map[string]*string{
		"PhoneNumber": tea.String(phone),
		"VerifyCode":  tea.String(code),
	}

	request := &openapi.OpenApiRequest{
		Query: query,
	}

	runtime := &util.RuntimeOptions{}

	resp, err := client.CallApi(params, request, runtime)
	if err != nil {
		return false, fmt.Errorf("CallApi failed: %v", err)
	}

	body, ok := resp["body"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("unexpected response body type")
	}

	if resCode, ok := body["Code"].(string); ok && resCode == "OK" {
		// Check Model.VerifyResult if available, but usually OK implies success for this API
		if model, ok := body["Model"].(map[string]interface{}); ok {
			if verifyResult, ok := model["VerifyResult"].(string); ok {
				return verifyResult == "PASS", nil
			}
		}
		return true, nil
	}

	msg, _ := body["Message"].(string)
	return false, fmt.Errorf("Verification failed: %s", msg)
}
