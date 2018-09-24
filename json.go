// Support JSON marshalling of objects.

package vtypes

import "encoding/json"

func (self *BaseObject) _getJsonValue() interface{} {
	switch self.parser.(type) {
	case Integerer:
		return self.AsInteger()
	case Stringer:
		return self.AsString()
	case Getter:
		res := make(map[string]interface{})
		for _, field := range self.Fields() {
			res[field] = self.Get(field).Value()
		}

		return res

	default:
		return self.DebugString()
	}
}

func (self *BaseObject) MarshalJSON() ([]byte, error) {
	buf, err := json.Marshal(self._getJsonValue())
	return buf, err
}
