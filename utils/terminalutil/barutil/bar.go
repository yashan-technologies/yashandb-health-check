package barutil

import (
	"fmt"
	"io"
	"strings"
	"time"

	"yhc/defs/bashdef"
	"yhc/i18n"

	mpb "github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

type bar struct {
	Name     string
	tasks    []*task
	bar      *mpb.Bar
	width    int
	progress *Progress
}

type barOption func(b *bar)

func withBarWidth(width int) barOption {
	return func(b *bar) {
		b.width = width
	}
}

func newBar(name string, progress *Progress, opts ...barOption) *bar {
	b := &bar{
		Name:     name,
		tasks:    make([]*task, 0),
		progress: progress,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func (b *bar) addTask(name string, worker func(string) error) {
	b.tasks = append(b.tasks, &task{
		name:   name,
		worker: worker,
		done:   make(chan struct{}),
	})
}

func (b *bar) draw() {
	barUnder := func(w io.Writer, s decor.Statistics) (err error) {
		for _, task := range b.tasks {
			if !task.finished {
				return
			}
			if task.err == nil {
				return
			}
			lines := b.genMsg(task.name, task.err)
			if err := b.printLine(w, lines); err != nil {
				return err
			}
		}
		return
	}
	bar := b.progress.mpbProgress.AddBar(int64(len(b.tasks)),
		mpb.BarExtender(mpb.BarFillerFunc(barUnder), false),
		mpb.PrependDecorators(
			// simple name decorator
			decor.Name(strings.ToUpper(b.Name)),
			// decor.DSyncWidth bit enables column width synchronization
			decor.Percentage(decor.WCSyncSpace),
		),
		mpb.AppendDecorators(
			decor.OnComplete(
				// ETA decorator with ewma age of 30
				decor.Name(i18n.T("progress.checking")),
				i18n.T("progress.done"),
			),
		),
	)
	b.bar = bar
}

func (b *bar) run() {
	defer b.progress.wg.Done()
	for _, t := range b.tasks {
		go func(t *task) {
			now := time.Now()
			t.start()
			t.wait()
			end := time.Now()
			b.bar.EwmaIncrement(end.Sub(now))
		}(t)
	}
	b.bar.Wait()
}

func (b *bar) splitMsg(msg string) []string {
	lines := make([]string, 0)
	b.cutWithMaxStep(msg, &lines)
	return lines
}

func (b *bar) cutWithMaxStep(str string, lines *[]string) {
	if len(str) <= b.width {
		*lines = append(*lines, str)
		return
	}
	index := b.getCutIndex(str)
	if index == 0 {
		return
	}
	*lines = append(*lines, str[:index])
	b.cutWithMaxStep(str[index:], lines)
}

func (b *bar) getCutIndex(str string) int {
	var length int
	for i := range str {
		if str[i] < 128 {
			length++
		} else {
			length += 2
		}
		if length > b.width {
			return i
		}
	}
	return 0
}

func (b *bar) genMsg(name string, err error) []string {
	var msg string
	if err == nil {
		msg = fmt.Sprintf("%s has been %s", name, bashdef.WithColor(i18n.T("progress.completed"), bashdef.COLOR_GREEN))
	} else {
		msg = fmt.Sprintf("%s has been %s err: %s", name, bashdef.WithColor(i18n.T("progress.failed"), bashdef.COLOR_RED), err.Error())
	}
	lines := b.splitMsg(msg)
	return lines
}

func (b *bar) printLine(w io.Writer, lines []string) error {
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "\t%s\n", line); err != nil {
			return err
		}
	}
	return nil
}
