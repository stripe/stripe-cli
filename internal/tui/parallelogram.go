package tui

import (
	"image/color"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"
)

const (
	paraWidth       = 60
	paraHeight      = 20
	animFPS         = 30
	dotChar         = "•"
	cellAspectRatio = 2.0 // terminal cells are ~2x taller than wide
	baseRadius      = 2.0
	mouseIdleTicks  = animFPS / 2
)

type dotState int

const (
	dotDim dotState = iota
	dotMid
	dotBright
)

type dot struct {
	x, y         int
	inShape      bool
	state        dotState
	flickerPhase float64
}

type animationFrameMsg struct {
	tag int
}

type parallelogram struct {
	width  int
	height int
	dots   []dot
	frame  int
	tag    int

	velocity    float64
	springVel   float64
	spring      harmonica.Spring
	idleTicks   int
	mouseActive bool
}

func newParallelogram(w, h int) parallelogram {
	skew := float64(h) * 0.30
	dots := make([]dot, 0, w*h)

	for x := range w {
		t := float64(x) / float64(w-1)
		topY := int(math.Round(skew * (1.0 - t)))
		botY := int(math.Round(float64(h-1) - skew*t))

		for y := range h {
			inShape := y >= topY && y <= botY
			dots = append(dots, dot{
				x:            x,
				y:            y,
				inShape:      inShape,
				state:        dotMid,
				flickerPhase: rand.Float64() * math.Pi * 2, //nolint:gosec
			})
		}
	}

	return parallelogram{
		width:  w,
		height: h,
		dots:   dots,
		spring: harmonica.NewSpring(harmonica.FPS(animFPS), 6.0, 1.0),
	}
}

func (p parallelogram) origin(screenW, screenH int) (int, int) {
	ox := (screenW - p.width) / 2
	oy := (screenH-p.height)/2 - 4
	return ox, oy
}

func (p parallelogram) tick() tea.Msg {
	return animationFrameMsg{tag: p.tag}
}

func (p *parallelogram) nextTick() tea.Cmd {
	tag := p.tag
	return tea.Tick(time.Second/animFPS, func(_ time.Time) tea.Msg {
		return animationFrameMsg{tag: tag}
	})
}

func (p *parallelogram) update(msg animationFrameMsg, mouse tea.Mouse, screenW, screenH int) (tea.Cmd, bool) {
	// tag acts as a sequence number: each frame increments it and the next tick
	// carries the new value, so outdated ticks (from before a resize, etc.) are
	// discarded rather than double-advancing the animation.
	if msg.tag > 0 && msg.tag != p.tag {
		return nil, false
	}

	p.frame++
	p.tag++
	p.idleTicks++

	p.velocity, p.springVel = p.spring.Update(p.velocity, p.springVel, 0)

	if p.idleTicks >= mouseIdleTicks {
		p.mouseActive = false
	}

	p.updateDots(mouse, screenW, screenH)
	return p.nextTick(), true
}

// addMotion registers mouse movement speed to grow the effect radius.
func (p *parallelogram) addMotion(dx, dy float64) {
	p.idleTicks = 0
	p.mouseActive = true
	p.velocity = math.Min(30, p.velocity+math.Sqrt(dx*dx+dy*dy)*0.5)
}

func (p *parallelogram) updateDots(mouse tea.Mouse, screenW, screenH int) {
	now := float64(p.frame) / animFPS
	effectRadius := baseRadius + p.velocity*0.8

	originX, originY := p.origin(screenW, screenH)

	for i := range p.dots {
		d := &p.dots[i]

		if !d.inShape {
			d.state = dotDim
			continue
		}

		if p.mouseActive {
			dotScreenX := originX + d.x
			dotScreenY := originY + d.y
			dx := float64(dotScreenX - mouse.X)
			dy := float64(dotScreenY-mouse.Y) * cellAspectRatio
			distToMouse := math.Sqrt(dx*dx + dy*dy)

			if distToMouse < effectRadius {
				proximity := 1.0 - distToMouse/effectRadius
				if rand.Float64() < 0.3+proximity*0.5 { //nolint:gosec
					d.state = dotBright
				} else {
					d.state = dotDim
				}
				continue
			}
		}

		pulse := math.Sin(now*1.8+d.flickerPhase*2) * 0.08
		wave := math.Sin(now*1.5+float64(d.x)*0.12+float64(d.y)*0.1) * 0.06
		level := 0.35 + pulse + wave

		switch {
		case level > 0.4:
			d.state = dotMid
		default:
			d.state = dotDim
		}
	}
}

func (p parallelogram) view(bright, mid, dim color.Color) string {
	grid := make([][]string, p.height)
	for i := range grid {
		grid[i] = make([]string, p.width)
		for j := range grid[i] {
			grid[i][j] = " "
		}
	}

	for _, d := range p.dots {
		if !d.inShape {
			continue
		}
		var shade color.Color
		switch d.state {
		case dotBright:
			shade = bright
		case dotMid:
			shade = mid
		case dotDim:
			shade = dim
		}
		grid[d.y][d.x] = lipgloss.NewStyle().Foreground(shade).Render(dotChar)
	}

	var sb strings.Builder
	for _, row := range grid {
		sb.WriteString(strings.Join(row, ""))
		sb.WriteByte('\n')
	}
	return sb.String()
}
