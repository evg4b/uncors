package render

import (
	"net/http"
	"strconv"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tui/styles"
	"github.com/evg4b/uncors/pkg/urlt"
)

const (
	requestPrefixWidth = 11
	methodWidth        = 5
	maxCachedBlocks    = 256
	inlineSample       = "x"
)

type blockKind uint8

const (
	statusBlock blockKind = iota
	cancelledBlock
	pendingBlock
)

type blockKey struct {
	kind blockKind
	code int
	text string
}

type inline struct {
	style       lipgloss.Style
	open, close string
}

func newInline(style lipgloss.Style) inline {
	open, closing, _ := strings.Cut(style.Render(inlineSample), inlineSample)

	return inline{style: style, open: open, close: closing}
}

func (i inline) render(text string) string {
	if text == "" || strings.ContainsAny(text, "\n\t") {
		return i.style.Render(text)
	}

	return i.open + text + i.close
}

type Formatter struct {
	theme     *styles.Theme
	cancelled lipgloss.Style
	pending   lipgloss.Style
	url       inline
	dim       inline

	mutex  sync.Mutex
	blocks map[blockKey]string
}

func NewFormatter(theme *styles.Theme) *Formatter {
	return &Formatter{
		theme:     theme,
		cancelled: theme.Cancelled.Width(requestPrefixWidth),
		pending:   theme.PendingBlock.Width(requestPrefixWidth),
		url:       newInline(theme.URL),
		dim:       newInline(theme.DimText),
		blocks:    make(map[blockKey]string, maxCachedBlocks),
	}
}

func (f *Formatter) Request(event events.RequestCompleted) string {
	data := event.Data
	method := methodLabel(data.Method)

	var builder strings.Builder

	builder.WriteString(f.theme.HandlerLabel(event.Handler))

	if data.Cancelled {
		builder.WriteString(f.block(blockKey{kind: cancelledBlock, text: "CXL " + method}))
	} else {
		key := blockKey{kind: statusBlock, code: data.Code, text: strconv.Itoa(data.Code) + " " + method}
		builder.WriteString(f.block(key))
	}

	builder.WriteByte(' ')
	builder.WriteString(f.url.render(urlt.URL_String(data.URL)))
	f.writeRewrite(&builder, event.Rewrite)

	if event.Err != nil && !data.Cancelled {
		builder.WriteString("\n  ")
		builder.WriteString(f.dim.render("↳ " + event.Err.Error()))
	}

	return builder.String()
}

func (f *Formatter) Pending(event events.RequestStarted, frame string) string {
	urlStr := ""
	if event.URL != nil {
		urlStr = urlt.URL_String(event.URL)
	}

	var builder strings.Builder

	builder.WriteString(f.theme.HandlerLabel(event.Handler))
	builder.WriteString(f.block(blockKey{kind: pendingBlock, text: frame + " " + methodLabel(event.Method)}))
	builder.WriteByte(' ')
	builder.WriteString(f.dim.render(urlStr))
	f.writeRewrite(&builder, event.Rewrite)

	return builder.String()
}

func (f *Formatter) writeRewrite(builder *strings.Builder, rewrite *events.Rewrite) {
	if rewrite != nil {
		builder.WriteByte(' ')
		builder.WriteString(f.dim.render("→ " + rewrite.To))
	}
}

func (f *Formatter) block(key blockKey) string {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if rendered, ok := f.blocks[key]; ok {
		return rendered
	}

	rendered := f.blockStyle(key).Render(key.text)
	if len(f.blocks) < maxCachedBlocks {
		f.blocks[key] = rendered
	}

	return rendered
}

func (f *Formatter) blockStyle(key blockKey) lipgloss.Style {
	switch key.kind {
	case cancelledBlock:
		return f.cancelled
	case pendingBlock:
		return f.pending
	case statusBlock:
	}

	return f.theme.StatusStyle(key.code).Width(requestPrefixWidth)
}

var methodLabels = map[string]string{
	http.MethodDelete:  "DEL",
	http.MethodOptions: "OPTS",
	http.MethodConnect: "CONN",
}

func methodLabel(method string) string {
	if label, ok := methodLabels[method]; ok {
		return label
	}

	return ansi.Truncate(method, methodWidth, "…")
}
