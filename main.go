package main

import (
	"bytes"
	"context"
	"fmt"
	cdp "github.com/chromedp/chromedp"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	goQrcode "github.com/skip2/go-qrcode"
	"image"
	"log"
	"strings"
	"time"
)

//抢购时间和支付密码，请注意，如果是00秒抢购，请在让Second == 60
const (
	//抢购的时间
	buyHour = 23
	buyMinute = 60
	buySecond = 00
	//支付密码
	passWord = `123456`
)

func main()  {
	ctx, _ := cdp.NewExecAllocator(
		context.Background(),

		// 以默认配置的数组为基础，覆写headless参数
		// 当然也可以根据自己的需要进行修改，这个flag是浏览器的设置
		append(
			cdp.DefaultExecAllocatorOptions[:],
			//如果选择 headless == true, 则会弹出chrome页面
			cdp.Flag(`headless`, false),
			//taobao存在找不到按钮问题
			//cdp.Flag("start-fullscreen", true),
		)...,
	)
	ctx, cancel := cdp.NewContext(ctx)

	defer cancel()

	urlStr := `https://login.taobao.com/`

	if err := cdp.Run(ctx, myTasks1(urlStr)); err != nil {
		log.Fatal(err)
		return
	}

	//临近抢购时间前，定时刷新页面
	for true {
		//判断是否已接近时间
		currentHour := time.Now().Hour()
		currentMinute := time.Now().Minute()
		judgement := (buyHour == currentHour) && (buyMinute - currentMinute <= 2)
		if judgement {
			break
		} else {
			if err := cdp.Run(ctx, myTasks2()); err != nil {
				log.Fatal(err)
				return
			}
		}
	}

	//进入最后激动人心的抢购时间
	for true {
		//判断是否已到抢购时间的秒
		currentSecond := time.Now().Second()
		if currentSecond == buySecond{
			if err := cdp.Run(ctx, myTasks3()); err != nil {
				log.Fatal(err)
				return
			}
		}
	}
}

//登录
func myTasks1(urlStr string) cdp.Tasks {
	return cdp.Tasks{
		cdp.Navigate(urlStr),
		cdp.Sleep(5 * time.Second),
		cdp.Click(`.icon-qrcode`),
		cdp.Sleep(5 * time.Second),
		getCode(),
		checkLoginStatus(),
		cdp.Navigate(`https://cart.taobao.com/cart.htm`),
		checkLoginCart(),
	}
}
//刷新页面，避免被自动logout
func myTasks2() cdp.Tasks {
	return cdp.Tasks{
		cdp.Navigate(`https://cart.taobao.com/cart.htm`),
		cdp.Sleep(60 * time.Second),
		//勾选购物车的所有物品
		cdp.Click(`#J_SelectAll1`),
	}
}
//结算订单
func myTasks3() cdp.Tasks {
	return cdp.Tasks{
		//提交结算
		cdp.Click(`#J_Go`),
		//提交订单
		cdp.WaitReady(`document.querySelector("#submitOrderPC_1 > div.wrapper > a.go-btn")`, cdp.ByJSPath),
		cdp.Click(`document.querySelector("#submitOrderPC_1 > div.wrapper > a.go-btn")`, cdp.ByJSPath),
		//输入付款密码
		cdp.WaitReady(`document.querySelector("#payPassword_rsainput")`, cdp.ByJSPath),
		cdp.SendKeys(`document.querySelector("#payPassword_rsainput")`, passWord, cdp.ByJSPath),
		//点击付款按钮
		cdp.Click(`J_authSubmit`, cdp.ByID),
		cdp.Sleep(10 * time.Second),
	}
}


//以下是辅助函数
//获取二维码
func getCode() cdp.ActionFunc {
	return func(ctx context.Context) (err error) {
		//_, _, contentSize, err := page.GetLayoutMetrics().Do(ctx)
		//if err != nil {
		//	return err
		//}
		//
		//width, height := int64(math.Ceil(contentSize.Width)), int64(math.Ceil(contentSize.Height))
		//
		//// force viewport emulation
		//err = emulation.SetDeviceMetricsOverride(width, height, 1, false).
		//	WithScreenOrientation(&emulation.ScreenOrientation{
		//		Type:  emulation.OrientationTypePortraitPrimary,
		//		Angle: 0,
		//	}).
		//	Do(ctx)
		//if err != nil {
		//	return err
		//}
		// 1. 用于存储图片的字节切片
		var code []byte

		// 2. 截图
		// 注意这里需要注明直接使用ID选择器来获取元素（chromedp.ByID）
		if err = cdp.Screenshot(`#content`,
			&code, cdp.ByID).Do(ctx); err != nil {
			return
		}

		//// 3. 保存文件
		//if err = ioutil.WriteFile("code.png", code, 0666); err != nil {
		//	return
		//}
		//return

		// 3. 把二维码输出到标准输出流
		if err = printQRCode(code); err != nil {
			fmt.Println(err)
			return err
		}
		return
	}
}
//打印二维码
func printQRCode(code []byte) (err error) {
	// 1. 因为我们的字节流是图像，所以我们需要先解码字节流
	img, _, err := image.Decode(bytes.NewReader(code))
	if err != nil {
		return
	}
	// 2. 然后使用gozxing库解码图片获取二进制位图
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return
	}
	// 3. 用二进制位图解码获取gozxing的二维码对象
	res, err := qrcode.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		return
	}
	// 4. 用结果来获取go-qrcode对象（注意这里我用了库的别名）
	qr, err := goQrcode.New(res.String(), goQrcode.Medium)
	if err != nil {
		return
	}
	// 5. 输出到标准输出流\
	fmt.Println("请扫描二维码登录淘宝账号")
	fmt.Println(qr.ToSmallString(false))
	return
}
// 检查是否登陆
func checkLoginStatus() cdp.ActionFunc {
	return func(ctx context.Context) (err error) {
		if err = cdp.WaitVisible(`#J_SiteNav`, cdp.ByID).Do(ctx); err != nil {
			return
		}
		var url string
		if err = cdp.Evaluate(`window.location.href`, &url).Do(ctx); err != nil {
			return
		}
		if strings.Contains(url, "https://i.taobao.com") {
			fmt.Println("已经使用二维码成功登录")
		}
		return
	}
}
//检查是否已经进入购物车
func checkLoginCart() cdp.ActionFunc {
	return func(ctx context.Context) (err error) {
		if err = cdp.WaitVisible(`#J_SelectAll1`, cdp.ByID).Do(ctx); err != nil {
			return
		}
		var url string
		if err = cdp.Evaluate(`window.location.href`, &url).Do(ctx); err != nil {
			return
		}
		if strings.Contains(url, "https://cart.taobao.com") {
			fmt.Println("已经成功进入购物车")
		}
		return
	}
}

