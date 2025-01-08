package pay

import (
	"context"
	"io/ioutil"
	"log"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"
	"github.com/quarkcloudio/quark-smart/v2/pkg/utils"
)

// 微信 api 字典：https://pay.weixin.qq.com/wiki/doc/apiv3/apis/index.shtml

type WechatPay struct {
	config *WechatPayConfig
	client *wechat.ClientV3
}

type WechatPayConfig struct {
	MchId          string // 商户ID 或者服务商模式的 sp_mchid
	SerialNo       string // 商户证书的证书序列号
	ApiV3Key       string // apiV3Key，商户平台获取
	PrivateKeyPath string // 私钥 apiclient_key.pem 文件路径
}

// 初始化微信支付客户端
//
// 如果使用默认配置首先需要在 configs 表中添加 name 为 WECHAT_PAY_MCH_ID、WECHAT_PAY_SERIAL_NO、WECHAT_PAY_API_V3_KEY、WECHAT_PAY_PRIVATE_KEY_PATH 的记录
func NewWechatPay(configs ...WechatPayConfig) *WechatPay {
	var config WechatPayConfig
	if len(configs) <= 0 {
		config = WechatPayConfig{
			MchId:          utils.GetConfig("WECHAT_PAY_MCH_ID"),
			SerialNo:       utils.GetConfig("WECHAT_PAY_SERIAL_NO"),
			ApiV3Key:       utils.GetConfig("WECHAT_PAY_API_V3_KEY"),
			PrivateKeyPath: utils.GetConfig("WECHAT_PAY_PRIVATE_KEY_PATH"),
		}
	} else {
		config = configs[0]
	}

	// 读取私钥内容
	privateKeyBytes, err := ioutil.ReadFile(config.PrivateKeyPath)
	if err != nil {
		log.Println("读取私钥文件失败：", err)
		return nil
	}

	// 初始化微信支付客户端
	client, err := wechat.NewClientV3(config.MchId, config.SerialNo, config.ApiV3Key, string(privateKeyBytes))
	if err != nil {
		log.Println("初始化微信支付客户端失败：", err)
		return nil
	}

	//启用自动同步返回验签，并定时更新微信平台API证书
	err = client.AutoVerifySign()
	if err != nil {
		log.Println("启用自动同步返回验签失败：", err)
		return nil
	}

	return &WechatPay{
		config: &config,
		client: client,
	}
}

// 微信 JSAPI 支付
//
// 具体传参请参考官方文档：https://pay.weixin.qq.com/wiki/doc/apiv3/apis/chapter3_1_1.shtml
func (p *WechatPay) JSAPIPay(param map[string]interface{}) (*wechat.JSAPIPayParams, error) {
	var bodyMap gopay.BodyMap
	for key, value := range param {
		bodyMap.Set(key, value)
	}

	// 拉起 JSAPI 支付
	perPayResponse, err := p.client.V3TransactionJsapi(context.Background(), bodyMap)
	if err != nil {
		return nil, err
	}
	if perPayResponse.Code != wechat.Success {
		return nil, err
	}

	// 获取拉起支付需要的 Pay Sign
	return p.client.PaySignOfJSAPI(utils.GetConfig("WECHAT_APP_ID"), perPayResponse.Response.PrepayId)
}

// 微信小程序支付
//
// 具体传参请参考官方文档：https://pay.weixin.qq.com/wiki/doc/apiv3/apis/chapter3_5_1.shtml
func (p *WechatPay) AppletPay(param map[string]interface{}) (*wechat.AppletParams, error) {
	var bodyMap gopay.BodyMap
	for key, value := range param {
		bodyMap.Set(key, value)
	}

	// 拉起 JSAPI 支付
	perPayResponse, err := p.client.V3TransactionJsapi(context.Background(), bodyMap)
	if err != nil {
		return nil, err
	}
	if perPayResponse.Code != wechat.Success {
		return nil, err
	}

	// 获取拉起支付需要的 Pay Sign
	return p.client.PaySignOfApplet(utils.GetConfig("WECHAT_APP_ID"), perPayResponse.Response.PrepayId)
}

// 微信 APP 支付
//
// 具体传参请参考官方文档：https://pay.weixin.qq.com/wiki/doc/apiv3/apis/chapter3_2_1.shtml
func (p *WechatPay) AppPay(param map[string]interface{}) (*wechat.AppPayParams, error) {
	var bodyMap gopay.BodyMap
	for key, value := range param {
		bodyMap.Set(key, value)
	}

	// 拉起 APP 支付
	perPayResponse, err := p.client.V3TransactionApp(context.Background(), bodyMap)
	if err != nil {
		return nil, err
	}
	if perPayResponse.Code != wechat.Success {
		return nil, err
	}

	// 获取拉起支付需要的 Pay Sign
	return p.client.PaySignOfApp(utils.GetConfig("WECHAT_APP_ID"), perPayResponse.Response.PrepayId)
}
