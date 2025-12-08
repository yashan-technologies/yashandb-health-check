package errdef

import (
	"errors"
	"fmt"

	"yhc/i18n"
)

var (
	ErrPermission    = errors.New(i18n.T("error.some_permission"))
	ErrExitWithCtrlC = errors.New(i18n.T("error.exit_with_ctrl_c"))
)

type FormItemUnFound struct {
	ItemName string
}

func NewFormItemUnFound(itemName string) *FormItemUnFound {
	return &FormItemUnFound{
		ItemName: itemName,
	}
}

func (e *FormItemUnFound) Error() string {
	return fmt.Sprintf(i18n.T("error.form_item_unfound"), e.ItemName)
}

type ItemEmpty struct {
	ItemName string
}

func NewItemEmpty(itemName string) *ItemEmpty {
	return &ItemEmpty{
		ItemName: itemName,
	}
}

func (e *ItemEmpty) Error() string {
	return fmt.Sprintf(i18n.T("error.item_empty"), e.ItemName)
}
