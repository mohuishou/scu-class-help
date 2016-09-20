package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	h "net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/axgle/mahonia"
)

func main() {

	var (
		uid, password, cid, sid string
		err                     error
		c                       *h.Client
		count                   int
	)
	//获取用户输入的学号，密码
	for {
		fmt.Println("请输入你的学号：")
		_, err = fmt.Scanln(&uid)
		fmt.Println("请输入你的密码：")
		_, err = fmt.Scanln(&password)
		if err != nil {
			mlog(err.Error())
			return
		}
		//模拟登录教务处，判断登录结果
		c, err = login(uid, password)
		if err != nil {
			mlog(err.Error())
			continue
		}
		mlog("教务处登录成功！")
		break

	}

	//todo:教务处登录成功之后开始发送选课请求
	fmt.Println("请输入待选择的课程号：")
	_, err = fmt.Scanln(&cid)
	fmt.Println("请输入待选择的课序号：")
	_, err = fmt.Scanln(&sid)
	count = 0
C:
	for i := 0; i < 2000; i++ {
		status, err := choose(cid, sid, c)
		if err != nil {
			mlog(err.Error())
		}
		count++
		switch status {
		case 4:
			mlog("已尝试" + strconv.Itoa(count) + "次，5秒后重新发送选课请求")
			time.Sleep(5 * 1000 * 1000 * 1000)
			continue C
		case 5:
			mlog("登录信息已失效，5秒后重新自动登录")
			time.Sleep(5 * 1000 * 1000 * 1000)
			mlog("重新登录中....")
			c, err = login(uid, password)
			if err != nil {
				mlog(err.Error())
			}
			mlog("教务处登录成功！")
			continue C
		case 6:
			fmt.Println("请输入待选择的课程号：")
			_, err = fmt.Scanln(&cid)
			fmt.Println("请输入待选择的课序号：")
			_, err = fmt.Scanln(&sid)
			count = 0
		default:
			break C
		}
	}

	//todo：判断选课状态，如果可以等待之后循环运行
	mlog("程序已发送2000次请求了，请休息一会儿吧")
}

func loginFor() {

}

//选课
//cid:课程号，sid：课序号，返回选课状态 1:选课成功，2上课时间冲突，3不满足学生系所要求，4没有课余量
func choose(cid, sid string, c *h.Client) (status int, err error) {
	kcid := cid + "_" + sid
	param := "kch=" + cid + "&cxkxh=" + sid + "&kcm=&skjs=&kkxsjc=&skxq=&skjc=&pageNumber=-2&preActionType=2&actionType=5"
	kbURL := "http://202.115.47.141/xkAction.do?actionType=6"

	cxURL := "http://202.115.47.141/xkAction.do?" + param
	url := "http://202.115.47.141/xkAction.do?preActionType=5&actionType=9&kcId=" + kcid
	_, err = chooseDo(kbURL, c)
	if err != nil {
		return
	}
	_, err = chooseDo(cxURL, c)
	if err != nil {
		return
	}

	resp, err := chooseDo(url, c)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	//抓取提示字符
	reg := regexp.MustCompile(`<font color="#990000">(.+)</font>`)
	result := reg.Find(body)
	if result == nil {
		err := errors.New("课程查询失败，请检查课程号:" + cid + "，课序号:" + sid + ",是否输入正确")
		return 6, err
	}
	//字符编码转换
	enc := mahonia.NewDecoder("gbk")
	results := string(result)
	results = enc.ConvertString(results)

	//课程提交状态。输出日志，返回status
	switch {
	case strings.Contains(results, "选课成功"):
		status = 1
	case strings.Contains(results, "时间冲突"):
		status = 2
	case strings.Contains(results, "学生系所的要求"):
		status = 3
	case strings.Contains(results, "课余量"):
		status = 4
	case strings.Contains(results, "登录后再使用"):
		status = 5

	}
	mlog(string(results))

	return
}

func chooseDo(url string, c *h.Client) (resp *h.Response, err error) {
	req, err := h.NewRequest("POST", url, strings.NewReader("name=cjb"))
	req = setHeader(req)
	resp, err = c.Do(req)
	return
}

//登录教务处，返回登录的cookie
//uid:学号，password密码
func login(uid, password string) (cookie *h.Client, err error) {
	c := &h.Client{}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return
	}
	c.Jar = jar
	loginURL := "http://202.115.47.141/loginAction.do?zjh=" + uid + "&mm=" + password
	resp, err := chooseDo(loginURL, c)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	//todo:检测是否登录成功！
	re, err := regexp.Match("<td class=\"errorTop\">.+</td>", body)
	if err != nil {
		return
	}
	if re {
		err = errors.New("教务处登录失败")
	}

	return c, err
}

//设置请求的header
func setHeader(req *h.Request) *h.Request {
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/50.0.2661.102 Safari/537.36")
	req.Header.Set("Accept", "text/javascript, text/html, application/xml, text/xml, */*")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-Forwarded-For", randIP())
	return req
}

//随机生成ip地址
func randIP() (ip string) {
	for i := 0; i < 4; i++ {
		ip += strconv.Itoa(rand.Intn(235))
	}
	return
}

func mlog(str string) {
	file, err := os.OpenFile("log.txt", os.O_APPEND, os.ModeAppend)
	if err != nil {
		file, err = os.Create("log.txt")
		if err != nil {
			log.Fatalln(err)
		}
	}
	//

	logger := log.New(file, "", log.LstdFlags)
	log.Println(str)
	logger.Println(str + "\r\n")
}
