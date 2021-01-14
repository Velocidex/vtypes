// Support JSON marshalling of objects.

package vtypes

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Velocidex/ordereddict"
)

func (self *StructDefinition) UnmarshalJSON(p []byte) error {
	var tmp []json.RawMessage
	if err := json.Unmarshal(p, &tmp); err != nil {
		return err
	}

	if len(tmp) != 3 {
		return errors.New("Struct Definition should be [name, size, fields]")
	}

	if err := json.Unmarshal(tmp[0], &self.Name); err != nil {
		return err
	}

	if err := json.Unmarshal(tmp[1], &self.Size); err != nil {
		if err := json.Unmarshal(tmp[1], &self.SizeExpression); err != nil {
			return err
		}
	}

	if err := json.Unmarshal(tmp[2], &self.Fields); err != nil {
		return fmt.Errorf("Decoding struct %v: %v", self.Name, err)
	}

	return nil
}

func (self *FieldDefinition) UnmarshalJSON(p []byte) error {
	var tmp []json.RawMessage
	if err := json.Unmarshal(p, &tmp); err != nil {
		return err
	}

	if len(tmp) != 3 && len(tmp) != 4 {
		return errors.New("Field Definition should be [name, offset, type, options?]")
	}

	if err := json.Unmarshal(tmp[0], &self.Name); err != nil {
		return err
	}
	if err := json.Unmarshal(tmp[1], &self.Offset); err != nil {
		if err := json.Unmarshal(tmp[1], &self.OffsetExpression); err != nil {
			return err
		}
	}
	if err := json.Unmarshal(tmp[2], &self.Type); err != nil {
		return err
	}

	if len(tmp) == 4 {
		self.Options = ordereddict.NewDict()
		if err := json.Unmarshal(tmp[3], &self.Options); err != nil {
			return err
		}
	}

	return nil
}
