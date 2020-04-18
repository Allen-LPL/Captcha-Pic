package controllers

import (
	"encoding/json"
	"github.com/astaxie/beego"
	"captcha-pic/mask"
	"crypto/md5"
    "encoding/hex"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/muesli/cache2go"
	"log"
	"time"
	"math/rand"
	"image/draw"
	"strconv"
	"math"
)

type CaptchaController struct {
	beego.Controller
}

type PictureInfo struct {
	Wall 	string		`json:"wall"`
	Piece 	string		`json:"piece"`
	Key 	string		`json:"key"`
	//Index 	string		`json:"index"`
	//Shuffle string		`json:"shuffle"`
    //OffsetX int         `json:"offsetX"`    // for DEBUG
	OffsetY int         `json:"offsetY"`
}

type ValidateResult struct {
	Success   int     `json:"success"`
    Diff      int     `json:"diff"`
}

type PictureInfoReturnMsg struct {
	Code 	int		`json:"Code"`
	Data 	PictureInfo
}

type ValidateReturnMsg struct {
	Code 	int		`json:"Code"`
	Data 	ValidateResult
}

var tcp = "127.0.0.1:6389"
var db   = int64(3)

// PictureController.Get
//func (c *CaptchaController) Get() {
//	c.TplName = "captcha.tpl"
//}

func (c *CaptchaController) GetPicturesInfo() {

	var key = c.Input().Get("key")
    var shuffle = c.Input().Get("shuffle")
    var index []rune = nil
	var f1 = ""
	var f2 = ""
    var offsetX = 0
	var offsetY = 40

	// connect redis
	client, err := redis.Dial("tcp", tcp)
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return
	}
	defer client.Close()
	_, err = client.Do("SELECT", "3")
	if err != nil {
		client.Close()
		return
	}

	cache := cache2go.Cache("captcha")
	// TODO: Load image from disk cache
	if key == "" {
		var c1, c2 draw.Image = nil, nil
		c1, c2, offsetX, offsetY, _ = mask.GetDefaultBackgroundAfterMask()

		rand.Seed(time.Now().UnixNano())
		var secret = fmt.Sprintf("%d%d", time.Now().UnixNano(), rand.Intn(100))
		h := md5.New()

		h.Write([]byte(secret)) // 需要加密的字符串为 123456

		cipherStr := h.Sum(nil)

		key = hex.EncodeToString(cipherStr)
		f1 = fmt.Sprintf("static/pictures/wall_%s.png", key)
		f2 = fmt.Sprintf("static/pictures/piece_%s.png", key)

		index = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

		c1, index = mask.ShuffleImage(c1, index, shuffle == "0")

		// add data
		cache.Add(key, 60*time.Second, offsetX)
		_, err = client.Do("SET", key, offsetX, "EX", "600")
		if err != nil {
			client.Close()
			fmt.Println("redis set failed:", err)
		}

		offsetX, err := cache.Value(key)
		if err == nil {
			fmt.Println("Found value in cache:", offsetX.Data())
		} else {
			fmt.Println("Error retrieving value from cache:", err)
		}

		mask.CreateImageFile(f1, c1)
		mask.CreateImageFile(f2, c2)
	} else {
		// for DEBUG
		f1 = fmt.Sprintf("static/pictures/wall_%s.png", key)
		f2 = fmt.Sprintf("static/pictures/piece_%s.png", key)
        index = []rune(c.Input().Get("index"))

		offsetX, err := cache.Value(key)
		if err == nil {
			fmt.Println("Found value in cache:", offsetX.Data())
		} else {
			fmt.Println("Error retrieving value from cache:", err)
		}
	}

	var pi PictureInfoReturnMsg
	pi = PictureInfoReturnMsg {
		Code: 200,
		Data: PictureInfo{
			Wall: f1,
			Piece: f2,
			Key: key,
			//Index: string(index),
			//Shuffle: shuffle,
			//OffsetX: offsetX,    // for DEBUG
			OffsetY: offsetY,
		},
	}

	c.Data["json"] = pi
	c.ServeJSON()
}

func (c *CaptchaController) Validate() {
    var offsetX = c.Input().Get("offsetX")
    var key = c.Input().Get("key")
	cache := cache2go.Cache("captcha")

	var vr ValidateReturnMsg
	vr = ValidateReturnMsg {
		Code: 201,
		Data: ValidateResult{
			Success: 0,
			Diff: -1,
		},
	}

	var x int64 = 0
	var err error
	if x, err = strconv.ParseInt(offsetX, 10, 32); err != nil {
		c.Data["json"] = vr
		c.ServeJSON()
		return
	}

	cachedOffsetX, err := cache.Value(key)
	if err != nil {
		c.Data["json"] = vr
		c.ServeJSON()
		return
	}

	var diff = int(x) - cachedOffsetX.Data().(int)
	if math.Abs(float64(diff)) < 10 {
		vr.Code = 200
		vr.Data.Success = 1
		vr.Data.Diff = diff
	}

	cache.Delete(key)
	c.Data["json"] = vr
	c.ServeJSON()
}

func (c *CaptchaController) ValidateTcp(offsetX string, key string) string {
	var vr ValidateReturnMsg
	vr = ValidateReturnMsg {
		Code: 201,
		Data: ValidateResult{
			Success: 0,
			Diff: -1,
		},
	}

	if offsetX == "" {
		return "X轴坐标不能为空"
	}
	if key == "" {
		return "key不能为空"
	}

	var x int64 = 0
	var err error
	if x, err = strconv.ParseInt(offsetX, 10, 32); err != nil {
		rs, err := json.Marshal(vr)
		if err != nil{
			log.Fatalln(err)
		}
		return string(rs)
	}

	// connect redis
	client, err := redis.Dial("tcp", tcp)
	if err != nil {
		return err.Error()
	}
	defer client.Close()
	_, err = client.Do("SELECT", "3")
	if err != nil {
		return err.Error()
	}

	cachedOffsetX, err := redis.Int(client.Do("GET", key))
	if err != nil {
		rs, err := json.Marshal(vr)
		if err != nil{
			log.Fatalln(err)
		}
		return string(rs)
	}

	var diff = int(x) - cachedOffsetX
	if math.Abs(float64(diff)) < 10 {
		vr.Code = 200
		vr.Data.Success = 1
		vr.Data.Diff = diff
	}

	// del redis key
	_, err = client.Do("DEL", key)
	if err != nil {
		rs, err := json.Marshal(vr)
		if err != nil{
			log.Fatalln(err)
		}
		return string(rs)
	}

	rs, err := json.Marshal(vr)
	if err != nil{
		log.Fatalln(err)
	}

	return string(rs)
	//c.ServeJSON()
}