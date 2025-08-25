package fsm

type optional[A any] struct {
	value A
	valid bool
}

type result[A any] struct {
	optional optional[A]
	panicked bool
}

// unarySupplierFunc is a function that doesn't take any arguments and returns a value of type R.
type unarySupplierFunc[R any] func() R

func tryUnarySupplier[R any](supply unarySupplierFunc[R]) (res result[R]) {
	defer func() {
		if r := recover(); r != nil {
			res.panicked = true
		}
	}()
	got := supply()
	res.optional = optional[R]{value: got, valid: true}
	return
}
