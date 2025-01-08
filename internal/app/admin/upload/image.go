package upload

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/quarkcloudio/quark-go/v3"
	"github.com/quarkcloudio/quark-go/v3/model"
	"github.com/quarkcloudio/quark-go/v3/service"
	"github.com/quarkcloudio/quark-go/v3/template/admin/component/message"
	"github.com/quarkcloudio/quark-go/v3/template/admin/upload"
	"github.com/quarkcloudio/quark-smart/v2/config"
	"github.com/quarkcloudio/quark-smart/v2/internal/dto/response"
)

type Image struct {
	upload.Template
}

// 初始化
func (p *Image) Init(ctx *quark.Context) interface{} {

	// 限制文件大小
	p.LimitSize = config.App.UploadImageSize

	// 限制文件类型
	p.LimitType = config.App.UploadImageType

	// 设置文件上传路径
	p.SavePath = config.App.UploadImageSavePath

	return p
}

// 初始化路由映射
func (p *Image) RouteInit() interface{} {
	p.GET("/api/admin/upload/:resource/getList", p.GetList)
	p.Any("/api/admin/upload/:resource/delete", p.Delete)
	p.POST("/api/admin/upload/:resource/crop", p.Crop)
	p.POST("/api/admin/upload/:resource/handle", p.Handle)
	p.POST("/api/admin/upload/:resource/base64Handle", p.HandleFromBase64)

	return p
}

// 获取文件列表n
func (p *Image) GetList(ctx *quark.Context) error {
	page := ctx.Query("page", "1")
	categoryId := ctx.Query("categoryId", "")
	name := ctx.Query("name", "")
	startDate := ctx.Query("createtime[0]", "")
	endDate := ctx.Query("createtime[1]", "")
	currentPage, _ := strconv.Atoi(page.(string))

	adminInfo, err := service.NewAuthService(ctx).GetAdmin()
	pictures, total, err := service.NewAttachmentService().GetListBySearch(
		adminInfo.Id,
		"IMAGE",
		categoryId,
		name,
		startDate,
		endDate,
		currentPage,
	)
	if err != nil {
		return ctx.JSON(200, message.Error(err.Error()))
	}

	pagination := map[string]interface{}{
		"defaultCurrent": 1,
		"current":        currentPage,
		"pageSize":       8,
		"total":          total,
	}

	categorys, err := service.NewAttachmentCategoryService().GetList(adminInfo.Id)
	if err != nil {
		return ctx.JSON(200, message.Error(err.Error()))
	}

	return ctx.JSON(200, message.Success("获取成功", "", map[string]interface{}{
		"pagination": pagination,
		"lists":      pictures,
		"categorys":  categorys,
	}))
}

// 图片删除
func (p *Image) Delete(ctx *quark.Context) error {
	data := map[string]interface{}{}
	json.Unmarshal(ctx.Body(), &data)
	if data["id"] == "" {
		return ctx.JSON(200, message.Error("参数错误！"))
	}

	err := service.NewAttachmentService().DeleteById(data["id"])
	if err != nil {
		return ctx.JSON(200, message.Error(err.Error()))
	}

	return ctx.JSON(200, message.Success("操作成功"))
}

// 图片裁剪
func (p *Image) Crop(ctx *quark.Context) error {
	var (
		result *quark.FileInfo
		err    error
	)

	data := map[string]interface{}{}
	if err := ctx.BodyParser(&data); err != nil {
		return ctx.JSON(200, message.Error(err.Error()))
	}
	if data["id"] == "" || data["file"] == "" {
		return ctx.JSON(200, message.Error("参数错误！"))
	}

	pictureInfo, err := service.NewAttachmentService().GetInfoById(data["id"])
	if err != nil {
		return ctx.JSON(200, message.Error(err.Error()))
	}
	if pictureInfo.Id == 0 {
		return ctx.JSON(200, message.Error("文件不存在"))
	}

	adminInfo, err := service.NewAuthService(ctx).GetAdmin()
	if err != nil {
		return ctx.JSON(200, message.Error(err.Error()))
	}

	limitW := ctx.Query("limitW", "")
	limitH := ctx.Query("limitH", "")

	files := strings.Split(data["file"].(string), ",")
	if len(files) != 2 {
		return ctx.JSON(200, message.Error("格式错误"))
	}

	fileData, err := base64.StdEncoding.DecodeString(files[1]) //成图片文件并把文件写入到buffer
	if err != nil {
		return ctx.JSON(200, message.Error(err.Error()))
	}

	limitSize := reflect.
		ValueOf(ctx.Template).
		Elem().
		FieldByName("LimitSize").Int()

	limitType := reflect.
		ValueOf(ctx.Template).
		Elem().
		FieldByName("LimitType").Interface()

	limitImageWidth := int(reflect.
		ValueOf(ctx.Template).
		Elem().
		FieldByName("LimitImageWidth").Int())

	if limitW.(string) != "" {
		getLimitImageWidth, err := strconv.Atoi(limitW.(string))
		if err == nil {
			limitImageWidth = getLimitImageWidth
		}
	}

	limitImageHeight := int(reflect.
		ValueOf(ctx.Template).
		Elem().
		FieldByName("LimitImageHeight").Int())

	if limitH.(string) != "" {
		getLimitImageWidth, err := strconv.Atoi(limitH.(string))
		if err == nil {
			limitImageWidth = getLimitImageWidth
		}
	}

	savePath := reflect.
		ValueOf(ctx.Template).
		Elem().
		FieldByName("SavePath").String()

	driver := reflect.
		ValueOf(ctx.Template).
		Elem().
		FieldByName("Driver").String()

	ossConfig := reflect.
		ValueOf(ctx.Template).
		Elem().
		FieldByName("OSSConfig").Interface()

	minioConfig := reflect.
		ValueOf(ctx.Template).
		Elem().
		FieldByName("MinioConfig").Interface()

	fileSystem := quark.
		NewStorage(&quark.StorageConfig{
			LimitSize:        limitSize,
			LimitType:        limitType.([]string),
			LimitImageWidth:  limitImageWidth,
			LimitImageHeight: limitImageHeight,
			Driver:           driver,
			OSSConfig:        ossConfig.(*quark.OSSConfig),
			MinioConfig:      minioConfig.(*quark.MinioConfig),
		}).
		Reader(&quark.File{
			Content: fileData,
		})

	// 上传前回调
	getFileSystem, fileInfo, err := ctx.Template.(interface {
		BeforeHandle(ctx *quark.Context, fileSystem *quark.FileSystem) (*quark.FileSystem, *quark.FileInfo, error)
	}).BeforeHandle(ctx, fileSystem)
	if err != nil {
		return ctx.JSON(200, message.Error(err.Error()))
	}
	if fileInfo != nil {
		extra := ""
		if fileInfo.Extra != nil {
			extraData, err := json.Marshal(fileInfo.Extra)
			if err == nil {
				extra = string(extraData)
			}
		}

		// 更新数据库
		service.NewAttachmentService().UpdateById(pictureInfo.Id, model.Attachment{
			Source: "ADMIN",
			Uid:    adminInfo.Id,
			Name:   fileInfo.Name,
			Type:   "IMAGE",
			Size:   fileInfo.Size,
			Ext:    fileInfo.Ext,
			Path:   fileInfo.Path,
			Url:    fileInfo.Url,
			Hash:   fileInfo.Hash,
			Extra:  extra,
			Status: 1,
		})
	}

	result, err = getFileSystem.
		WithImageExtra().
		FileName(pictureInfo.Name).
		Path(savePath).
		Save()
	if err != nil {
		return ctx.JSON(200, message.Error(err.Error()))
	}

	// 重写url
	if driver == quark.LocalStorage {
		result.Url = service.NewAttachmentService().GetImagePath(result.Url)
	}

	extra := ""
	if result.Extra != nil {
		extraData, err := json.Marshal(result.Extra)
		if err == nil {
			extra = string(extraData)
		}
	}

	// 更新数据库
	service.NewAttachmentService().UpdateById(pictureInfo.Id, model.Attachment{
		Source: "ADMIN",
		Uid:    adminInfo.Id,
		Name:   result.Name,
		Type:   "IMAGE",
		Size:   result.Size,
		Ext:    result.Ext,
		Path:   result.Path,
		Url:    result.Url,
		Hash:   result.Hash,
		Extra:  extra,
		Status: 1,
	})

	return ctx.JSON(200, message.Success("操作成功", "", result))
}

// 上传前回调
func (p *Image) BeforeHandle(ctx *quark.Context, fileSystem *quark.FileSystem) (*quark.FileSystem, *quark.FileInfo, error) {
	fileHash, err := fileSystem.GetFileHash()
	if err != nil {
		return fileSystem, nil, err
	}

	imageInfo, err := service.NewAttachmentService().GetInfoByHash(fileHash)
	if err != nil {
		return fileSystem, nil, err
	}
	if imageInfo.Id != 0 {
		var extra map[string]interface{}
		if imageInfo.Extra != "" {
			_ = json.Unmarshal([]byte(imageInfo.Extra), &extra)
		}

		fileInfo := &quark.FileInfo{
			Name:  imageInfo.Name,
			Size:  imageInfo.Size,
			Ext:   imageInfo.Ext,
			Path:  imageInfo.Path,
			Url:   imageInfo.Url,
			Hash:  imageInfo.Hash,
			Extra: extra,
		}

		return fileSystem, fileInfo, err
	}

	return fileSystem, nil, err
}

// 上传完成后回调
func (p *Image) AfterHandle(ctx *quark.Context, result *quark.FileInfo) error {
	driver := reflect.
		ValueOf(ctx.Template).
		Elem().
		FieldByName("Driver").String()

	// 重写url
	if driver == quark.LocalStorage {
		result.Url = service.NewAttachmentService().GetImagePath(result.Url)
	}

	adminInfo, err := service.NewAuthService(ctx).GetAdmin()
	if err != nil {
		return ctx.JSON(200, message.Error(err.Error()))
	}

	extra := ""
	if result.Extra != nil {
		extraData, err := json.Marshal(result.Extra)
		if err == nil {
			extra = string(extraData)
		}
	}

	// 插入数据库
	id, err := service.NewAttachmentService().InsertGetId(model.Attachment{
		Source: "ADMIN",
		Uid:    adminInfo.Id,
		Name:   result.Name,
		Type:   "IMAGE",
		Size:   result.Size,
		Ext:    result.Ext,
		Path:   result.Path,
		Url:    result.Url,
		Hash:   result.Hash,
		Extra:  extra,
		Status: 1,
	})

	if err != nil {
		return ctx.JSON(200, message.Error(err.Error()))
	}

	return ctx.JSON(200, message.Success("上传成功", "", response.UploadResp{
		Id:          id,
		ContentType: result.ContentType,
		Ext:         result.Ext,
		Hash:        result.Hash,
		Name:        result.Name,
		Path:        result.Path,
		Size:        result.Size,
		Url:         result.Url,
		Extra:       result.Extra,
	}))
}
