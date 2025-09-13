package ali

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"one-api/dto"
	"one-api/relay/channel"
	"one-api/relay/channel/claude"
	"one-api/relay/channel/openai"
	relaycommon "one-api/relay/common"
	"one-api/relay/constant"
	"one-api/types"
	"strings"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.ClaudeRequest) (any, error) {
	return req, nil
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	var fullRequestURL string
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		fullRequestURL = fmt.Sprintf("%s/api/v2/apps/claude-code-proxy/v1/messages", info.ChannelBaseUrl)
	default:
		switch info.RelayMode {
		case constant.RelayModeEmbeddings:
			fullRequestURL = fmt.Sprintf("%s/compatible-mode/v1/embeddings", info.ChannelBaseUrl)
		case constant.RelayModeRerank:
			fullRequestURL = fmt.Sprintf("%s/api/v1/services/rerank/text-rerank/text-rerank", info.ChannelBaseUrl)
		case constant.RelayModeImagesGenerations:
			fullRequestURL = fmt.Sprintf("%s/api/v1/services/aigc/text2image/image-synthesis", info.ChannelBaseUrl)
		case constant.RelayModeImagesEdits:
			fullRequestURL = fmt.Sprintf("%s/api/v1/services/aigc/multimodal-generation/generation", info.ChannelBaseUrl)
		case constant.RelayModeCompletions:
			fullRequestURL = fmt.Sprintf("%s/compatible-mode/v1/completions", info.ChannelBaseUrl)
		default:
			fullRequestURL = fmt.Sprintf("%s/compatible-mode/v1/chat/completions", info.ChannelBaseUrl)
		}
	}

	return fullRequestURL, nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", "Bearer "+info.ApiKey)
	if info.IsStream {
		req.Set("X-DashScope-SSE", "enable")
	}
	if c.GetString("plugin") != "" {
		req.Set("X-DashScope-Plugin", c.GetString("plugin"))
	}
	if info.RelayMode == constant.RelayModeImagesGenerations {
		req.Set("X-DashScope-Async", "enable")
	}
	if info.RelayMode == constant.RelayModeImagesEdits {
		req.Set("Content-Type", "application/json")
	}
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	// docs: https://bailian.console.aliyun.com/?tab=api#/api/?type=model&url=2712216
	// fix: InternalError.Algo.InvalidParameter: The value of the enable_thinking parameter is restricted to True.
	if strings.Contains(request.Model, "thinking") {
		request.EnableThinking = true
		request.Stream = true
		info.IsStream = true
	}
	// fix: ali parameter.enable_thinking must be set to false for non-streaming calls
	if !info.IsStream {
		request.EnableThinking = false
	}

	switch info.RelayMode {
	default:
		aliReq := requestOpenAI2Ali(*request)
		return aliReq, nil
	}
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	if info.RelayMode == constant.RelayModeImagesGenerations {
		aliRequest, err := oaiImage2Ali(request)
		if err != nil {
			return nil, fmt.Errorf("convert image request failed: %w", err)
		}
		return aliRequest, nil
	} else if info.RelayMode == constant.RelayModeImagesEdits {
		// ali image edit https://bailian.console.aliyun.com/?tab=api#/api/?type=model&url=2976416
		// 如果用户使用表单，则需要解析表单数据
		if strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
			aliRequest, err := oaiFormEdit2AliImageEdit(c, info, request)
			if err != nil {
				return nil, fmt.Errorf("convert image edit form request failed: %w", err)
			}
			return aliRequest, nil
		} else {
			aliRequest, err := oaiImage2Ali(request)
			if err != nil {
				return nil, fmt.Errorf("convert image request failed: %w", err)
			}
			return aliRequest, nil
		}
	}
	return nil, fmt.Errorf("unsupported image relay mode: %d", info.RelayMode)
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return ConvertRerankRequest(request), nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	// TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		if info.IsStream {
			return claude.ClaudeStreamHandler(c, resp, info, claude.RequestModeMessage)
		} else {
			return claude.ClaudeHandler(c, resp, info, claude.RequestModeMessage)
		}
	default:
		switch info.RelayMode {
		case constant.RelayModeImagesGenerations:
			err, usage = aliImageHandler(c, resp, info)
		case constant.RelayModeImagesEdits:
			err, usage = aliImageEditHandler(c, resp, info)
		case constant.RelayModeRerank:
			err, usage = RerankHandler(c, resp, info)
		default:
			adaptor := openai.Adaptor{}
			usage, err = adaptor.DoResponse(c, resp, info)
		}
		return usage, err
	}
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
