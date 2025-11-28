package errdef

import (
	"errors"
	"fmt"

	"yhc/i18n"
)

var (
	ErrEndLessStart        = errors.New(i18n.T("error.end_less_start"))
	ErrStartShouldLessCurr = errors.New(i18n.T("error.start_should_less_curr"))
)

type ErrGreaterMaxDuration struct {
	MaxDuration string
}

func NewGreaterMaxDur(max string) *ErrGreaterMaxDuration {
	return &ErrGreaterMaxDuration{MaxDuration: max}
}

func (e ErrGreaterMaxDuration) Error() string {
	return fmt.Sprintf(i18n.T("error.greater_max_duration"), e.MaxDuration)
}

type ErrLessMinDuration struct {
	MinDuration string
}

func NewLessMinDur(min string) *ErrLessMinDuration {
	return &ErrLessMinDuration{MinDuration: min}
}

func (e ErrLessMinDuration) Error() string {
	return fmt.Sprintf(i18n.T("error.less_min_duration"), e.MinDuration)
}
