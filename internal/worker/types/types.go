package workertypes

type Result interface {
	Result() interface{}
	Errors() []error
}
