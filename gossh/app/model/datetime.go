package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

const (
	TimeFormat = "2006-01-02 15:04:05"
)

type DateTime time.Time

func NewDateTime(str string) (DateTime, error) {
	now, err := time.ParseInLocation(TimeFormat, str, time.Local)
	if err != nil {
		return DateTime{}, fmt.Errorf("can not convert %v to date,must like format:yyyy-MM-dd HH:mm:ss,simple example : %v", str, TimeFormat)
	}
	t := DateTime(now)
	return t, nil
}

func (t DateTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, time.Time(t).Format(TimeFormat))), nil
}

func (t *DateTime) UnmarshalJSON(b []byte) error {
	now, err := time.ParseInLocation(`"`+TimeFormat+`"`, string(b), time.Local)
	if err != nil {
		return fmt.Errorf("can not convert %v to date,must like format:yyyy-MM-dd HH:mm:ss,simple example : %v", string(b), TimeFormat)
	}
	*t = DateTime(now)
	return nil
}
func (t DateTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if time.Time(t).UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return time.Time(t), nil
}
func (t *DateTime) Scan(v any) error {
	value, ok := v.(time.Time)
	if ok {
		*t = DateTime(value)
		return nil
	}
	return fmt.Errorf("can not convert %v to timestamp", v)
}

func (t *DateTime) ToTime() time.Time {
	return time.Time(*t)
}

func (t DateTime) String() string {
	return time.Time(t).Format(TimeFormat)
}
