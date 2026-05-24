package validation
import (
	"github.com/go-playground/validator/v10"
)
// ValidatorClient is a type alias for the validator.Validate struct
type ValidatorClient = *validator.Validate
var validatorClient *validator.Validate

// Validable is an interface that requires a Validate method which takes a ValidatorClient and returns an error.
type Validable interface {
	Validate(ValidatorClient) error
}	

// init initializes the validator client when the package is imported.
func init() {
	validatorClient = validator.New()
}
// Validate takes a Validable payload and calls its Validate method, passing the validator client. 
// It returns any error that occurs during validation.
func Validate(payload Validable)error{
	err:= payload.Validate(validatorClient)
	isSuccess,err:= TransformValidationError(err)
	if isSuccess {
		return err
	}else{
		return err
	}
}