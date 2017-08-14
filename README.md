# Mimir
Use FFmpeg to edit Video

## API

API Version: /v1

Methods|API| USAGE| PARAMETER|RESPONSE|
-------|----|-----|----------|--------|
GET| / | Test Connect| --- | String |
POST|/video/shot| Screen Shot | Shot Object | ShotRespon Object |
POST|/video/fps| Change video fps | Fps Object | FpsRespon Object |
GET|/video/fps/:token| Get video change progress | --- | FpsRespon Object |
POST|/video/ratio| Get Video Ratio  | Ratio Object | RatioRespon Object |
PUT|/video/ratio| Moidfy Video Ratio  | Ratio Object | RatioRespon Object |
POST|/video/concat| Concat Multiple Videos | Concat Object | ConcatRespon Object|
GET|/video/concat/:token| Get Concat Progress | --- | ConcatRespon Object|
POST|/video/logo| Add Logo In Video | Logo Object | LogoRespon Object|
GET|/video/logo/:token| Get Logo Progress | --- | LogoRespon Object|
POST|/video/cut| Cut Video | Cut Object | CutRespon Object|
GET|/video/audio/:token| Get Audio Merge Progress | --- | AudioRespon Object|
POST|/video/audio| Sperate Audio From Video | Audio Object | AudioRespon Object|
PUT|/video/audio| Merge Audio From Video | Audio Object | AudioRespon Object|
POST|/video/separate| Sperate Video | Audio Object | AudioRespon Object|
GET|/video/black/:token| Get Black Add Progress | --- | BlackRespon Object|
POST|/video/black| Add Black In Vido | Black Object | BlackRespon Object|
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
                    // 当类型为2时，此处填写生成的图片个数
    "outformat":string  //输出的图片类型
	                    // 当类型为0时，此处为输出的图片名称
	                    // 当类型为1/2时，按照以下格式赋值 图片前缀|类型后缀 e.g. img|jpg 以img为前缀的jpg类型图片
    "size":string //输出指定尺寸的图片 e.g. 320*180
}
```

### ShotRespon Object
```
{
    "img": []string // 生成的截图文件名
}
```

### Fps Object
```
{
    "video":<ffmpeg object>
    "fps":int // 准备转变的帧率
}
```

### FpsRespon Object
```
{
    "fps":int // 视频原始帧率,当为查询请求时，此属性为0
    "token":string // 视频唯一标示，用于查询进度使用
    "progress":float32 // 视频变帧进度，创建变帧请求时返回0， 当token不存在时返回-1，正常变帧时返回当前进度
}

```

### Ratio Object
```
{
    "video":<ffmpeg object>,
    "ratio": string //当修改分辨率时，此属性必填。填写准备要转换的分辨率，e.g. 1280x720
}
```

### RatioRespon Object
```
{
    "ratio": string //视频分辨率
    "token": string //视频唯一标示
    "progress": float32 //分辨率转换进度
}
```

### Concat Object
```
{
    "input":[]string // 源文件全路径，将按照文件顺序依次合并
    "output": string // 输出文件全路径,不得为空
}
```

### RatioRespon Object
```
{
    "token": string //视频唯一标示
    "progress": float32 //合并进度
}
```

### Logo Object
```
{
    "video":<ffmpeg object>,
    "logo": string // logo文件全路径
    "postion", string // logo坐标位置,默认为00:00. 
}
```

### LogoRespon Object
```
{
    "token": string //视频唯一标示
    "progress": float32 //Logo添加进度
}
```

### Cut Object
```
{
    "video":<ffmpeg object>,
    "num": int // 切割视频个数,默认为5个
    "inter", string // 视频长度, mm:ss格式.默认为3秒 00:03
}
```

### CutRespon Object
```
{
    "output": []string //切割后的视频文件路径
}
```

### Audio Object
```
{
    "video":<ffmpeg object>,
    "audio":string // 当准备将视频和音频进行合并时，此属性为必填项
}
```

### AudioResponse Object
```
{
    "audio":string //分离的音频/视频流文件路径
    "token":string //视频唯一标识码
    "progress":float32 //合并操作进度
}
```

### Black Object
```
{
    "video":<ffmpeg object>
}
```
### BlackResponse Object
```
{
    "token":string //视频唯一标识码
    "progress":float32 //合并操作进度
    "output": string //输出的文件路径
}
```

## Error Code

Code|Value|
-------|----|
10001|读取Body出错|
10002|Body转换为Json出错|
10003|FFmpeg命令支持失败|
10004|Json转换为Body出错|
10005|视频变帧出错|
10006|输入文件源为空|
10007|输出文件为空|
10008|获取视频长度错误|
10009|视频合并数量大于5个|
10010|音频流文件不得为空|