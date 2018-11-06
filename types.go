package fsgraph

import (
	"fmt"
	io "io"

	"github.com/pkg/errors"
)

type Int64 int64

func (x *Int64) UnmarshalGQL(v interface{}) error {
	switch v := v.(type) {
	case int64:
		*x = Int64(v)
	case float64:
		*x = Int64(v)
	/*case string:
	y, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return err
	}
	*x = Int64(y)
	*/
	default:
		//spew.Dump(v)
		return errors.New("Invalid type for Int64")
	}
	return nil
}

func (x Int64) MarshalGQL(w io.Writer) {
	fmt.Fprintf(w, `%d`, x)
}
