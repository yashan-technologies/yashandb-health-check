package errdef

import (
	"errors"
	"fmt"

	"yhc/i18n"
)

var (
	ErrPathFormat = errors.New(i18n.T("error.path_format"))
)

type ErrPermissionDenied struct {
	User     string
	FileName string
}

type ErrFileNotFound struct {
	FName string
}

type ErrFileParseFailed struct {
	FName string
	Err   error
}

func NewErrPermissionDenied(user string, path string) *ErrPermissionDenied {
	return &ErrPermissionDenied{
		User:     user,
		FileName: path,
	}
}

func (e *ErrPermissionDenied) Error() string {
	return fmt.Sprintf(i18n.T("error.permission_denied"), e.User, e.FileName)
}

func (e *ErrFileNotFound) Error() string {
	return fmt.Sprintf(i18n.T("error.file_not_found"), e.FName)
}

func (e *ErrFileParseFailed) Error() string {
	return fmt.Sprintf(i18n.T("error.file_parse_failed"), e.FName, e.Err)
}
