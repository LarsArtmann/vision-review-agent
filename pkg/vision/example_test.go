package vision_test

import (
	"context"
	"errors"
	"fmt"

	"charm.land/fantasy"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

// exampleModel is the smallest possible fantasy.LanguageModel; validation
// never invokes it.
type exampleModel struct{}

func (exampleModel) Generate(_ context.Context, _ fantasy.Call) (*fantasy.Response, error) {
	return nil, errors.New("unused")
}

func (exampleModel) Provider() string { return "example" }

func (exampleModel) Model() string { return "example-model" }

func (exampleModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	return nil, errors.New("unused")
}

func (exampleModel) GenerateObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (*fantasy.ObjectResponse, error) {
	return nil, errors.New("unused")
}

func (exampleModel) StreamObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (fantasy.ObjectStreamResponse, error) {
	return nil, errors.New("unused")
}

// Config.Validate wraps its sentinels with the offending value, so errors.Is
// still matches the sentinel while the message stays self-diagnosing.
func ExampleConfig_validate() {
	err := (&vision.Config{Model: exampleModel{}, Temperature: 3.5}).Validate()

	fmt.Println(err)
	fmt.Println(errors.Is(err, vision.ErrInvalidTemperature))
	// Output:
	// vision agent: temperature must be between 0.0 and 2.0: got 3.50, want [0.0, 2.0]
	// true
}
