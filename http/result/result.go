package resx

import (
	"github.com/time-origin/warpin-go-common/errors"
)

type Result struct {
	Code  errx.ErrCode `json:"code"`  // 结果码
	Data  interface{}  `json:"data"`  // 结果集
	Count int          `json:"count"` //结果总数
	Msg   string       `json:"msg"`   // 结果描述
}

// NewResult creates a new success result object.
func NewResult(data interface{}, errCode errx.ErrCode, m ...string) *Result {
	r := &Result{Data: data, Code: errCode}
	if _, ok := data.(error); ok {
		if m == nil {
			r.Msg = errx.MapErrMsg(errCode)
		}
	} else {
		r.Msg = errx.MapErrMsg(errCode)
	}
	if len(m) > 0 {
		r.Msg = m[0]
	}

	return r
}

// NewResultWithCount creates a new success result object with a total count.
func NewResultWithCount(data interface{}, count int, errCode errx.ErrCode, m ...string) *Result {
	r := &Result{Data: data, Code: errCode, Count: count}
	if _, ok := data.(error); ok {
		if m == nil {
			r.Msg = errx.MapErrMsg(errCode)
		}
	} else {
		r.Msg = errx.MapErrMsg(errCode)
	}
	if len(m) > 0 {
		r.Msg = m[0]
	}

	return r
}

// NewFailResult creates a new failure result object.
func NewFailResult(errCode errx.ErrCode, m ...string) *Result {
	r := &Result{Code: errCode}
	if len(m) > 0 {
		if m[0] == "" {
			r.Msg = errx.MapErrMsg(errCode)
		} else {
			r.Msg = m[0]
		}
	} else {
		mes := errx.MapErrMsg(errCode)
		if mes == "" {
			r.Msg = errx.MapErrMsg(errx.ServerCommonError)
		} else {
			r.Msg = mes
		}

	}

	return r
}

// NewPageResult creates a new paginated result object.
func NewPageResult(data interface{}, errCode errx.ErrCode, count int, m ...string) *Result {
	r := &Result{Data: data, Code: errCode, Count: count}

	if _, ok := data.(error); ok {
		if m == nil {
			r.Msg = errx.MapErrMsg(errCode)
		}
	} else {
		r.Msg = errx.MapErrMsg(errCode)
	}
	if len(m) > 0 {
		r.Msg = m[0]
	}

	return r
}
