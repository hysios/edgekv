package edgekv

import "github.com/hysios/utils/errors"

var (
	ErrNonimpement = errors.New("nonimplement")
)

func init() {
	errors.RegisterErrCode(ErrNonimpement, errors.ErrAuto) // 10000
}
