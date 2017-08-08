# Mimir
Use FFmpeg to edit Video

## API

API Version: /v1

Methods|API| USAGE| PARAMETER|RESPONSE|
-------|----|-----|----------|--------|
GET| / | Test Connect| --- | String |
POST|/video/shot| Screen Shot | Shot Object | ShotRespon Object |

## API Object

### ffmpeg Object
```
{
    "input":string // 源文件
    "output":string // 输出文件
}
```

### Shot Object
```
{
    "video":<ffmpeg object> ,
    "type": int //截图类型 0:截取指定时间点 1:按间隔秒数截取 2:截取指定数量
    "value": string //类型值
	                // 当类型为0时，此处按照hh:mm:ss 填写时间
	                // 当类型为1时, 此处填写秒数
    "outformat":string  //输出的图片类型
	                    // 当类型为0时，此处为输出的图片名称
	                    // 当类型为1时，按照以下格式赋值 图片前缀|类型后缀 e.g. img|jpg 以img为前缀的jpg类型图片
    "size":string //输出指定尺寸的图片 e.g. 320*180
}
```

### ShotRespon Object
{
    "img": []string // 生成的截图文件名
}

## Error Code

Code|Value|
-------|----|
10001|读取Body出错|
10002|Body转换为Json出错|
10003|FFmpeg命令支持失败|
10004|Json转换为Body出错|