package tools

import (
	"context"
	"log"

	"charm.land/fantasy"
	"github.com/chromedp/chromedp"
)

type WebBrowserTool struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func (w *WebBrowserTool) Tools() []fantasy.AgentTool {
	type WebBrowserToolInput struct {
		URL string `json:"url"`
	}
	return []fantasy.AgentTool{
		fantasy.NewAgentTool[WebBrowserToolInput](
			"web_browser",
			"browse the web and retrieve information",
			func(ctx context.Context, input WebBrowserToolInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {

				log.Println("browser tool called", input.URL)
				var result string
				var buf []byte
				err := chromedp.Run(w.ctx,
					chromedp.Navigate(input.URL),
					chromedp.Text("body", &result, chromedp.NodeVisible, chromedp.ByQuery),
					chromedp.CaptureScreenshot(&buf),
				)
				if err != nil {
					return fantasy.NewTextResponse(err.Error()), err
				}

				return fantasy.NewImageResponse(buf, "image/png"), nil
			},
		),
	}
}

func NewWebBrowserTool() *WebBrowserTool {
	ctx, cancelFunc := chromedp.NewContext(context.Background())
	return &WebBrowserTool{
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}
}
