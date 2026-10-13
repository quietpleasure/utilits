package utilits

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

// Техномир присылает числа то числом, то строкой, то прочерком, поэтому
// парсеры разбирают данные вручную. Тест фиксирует все встречающиеся формы,
// чтобы переход с json.Unmarshal на strconv не изменил поведение.

func TestCustomIntUnmarshal(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{in: `5`, want: 5},
		{in: `"5"`, want: 5},
		{in: `0`, want: 0},
		{in: `-3`, want: -3},
		{in: `"-3"`, want: -3},
		{in: `"-"`, want: 0},
		{in: `""`, want: 0},
		{in: `null`, want: 0},
		{in: `"abc"`, wantErr: true},
		{in: `1.5`, wantErr: true},
	}
	for _, tc := range cases {
		var v CustomInt
		err := json.Unmarshal([]byte(tc.in), &v)
		if tc.wantErr {
			if err == nil {
				t.Errorf("CustomInt(%s): ожидалась ошибка, получено %d", tc.in, v.Int)
			}
			continue
		}
		if err != nil {
			t.Errorf("CustomInt(%s): неожиданная ошибка %v", tc.in, err)
			continue
		}
		if v.Int != tc.want {
			t.Errorf("CustomInt(%s) = %d, ожидалось %d", tc.in, v.Int, tc.want)
		}
	}
}

func TestCustomInt64Unmarshal(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{in: `9007199254740993`, want: 9007199254740993},
		{in: `"42"`, want: 42},
		{in: `"-"`, want: 0},
		{in: `""`, want: 0},
		{in: `null`, want: 0},
		{in: `"x"`, wantErr: true},
	}
	for _, tc := range cases {
		var v CustomInt64
		err := json.Unmarshal([]byte(tc.in), &v)
		if tc.wantErr {
			if err == nil {
				t.Errorf("CustomInt64(%s): ожидалась ошибка, получено %d", tc.in, v.Int64)
			}
			continue
		}
		if err != nil {
			t.Errorf("CustomInt64(%s): неожиданная ошибка %v", tc.in, err)
			continue
		}
		if v.Int64 != tc.want {
			t.Errorf("CustomInt64(%s) = %d, ожидалось %d", tc.in, v.Int64, tc.want)
		}
	}
}

func TestCustomFloat64Unmarshal(t *testing.T) {
	cases := []struct {
		in      string
		want    float64
		wantErr bool
	}{
		{in: `1.25`, want: 1.25},
		{in: `"1.25"`, want: 1.25},
		{in: `10`, want: 10},
		{in: `"1,234.56"`, want: 1234.56}, // разделитель тысяч вырезается
		{in: `"-"`, want: 0},
		{in: `""`, want: 0},
		{in: `null`, want: 0},
		{in: `"1.2.3"`, wantErr: true},
	}
	for _, tc := range cases {
		var v CustomFloat64
		err := json.Unmarshal([]byte(tc.in), &v)
		if tc.wantErr {
			if err == nil {
				t.Errorf("CustomFloat64(%s): ожидалась ошибка, получено %v", tc.in, v.Float64)
			}
			continue
		}
		if err != nil {
			t.Errorf("CustomFloat64(%s): неожиданная ошибка %v", tc.in, err)
			continue
		}
		if v.Float64 != tc.want {
			t.Errorf("CustomFloat64(%s) = %v, ожидалось %v", tc.in, v.Float64, tc.want)
		}
	}
}

func TestCustomTimeUnmarshal(t *testing.T) {
	var v CustomTime
	if err := json.Unmarshal([]byte(`"2026-09-03 14:05:06"`), &v); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	want := time.Date(2026, 9, 3, 14, 5, 6, 0, time.UTC)
	if !time.Time(v).Equal(want) {
		t.Errorf("CustomTime = %v, ожидалось %v", time.Time(v), want)
	}

	// Прочерк и неизвестный формат дают нулевое время, а не ошибку.
	for _, in := range []string{`"-"`, `"31/12/2026"`} {
		var z CustomTime
		if err := json.Unmarshal([]byte(in), &z); err != nil {
			t.Errorf("CustomTime(%s): неожиданная ошибка %v", in, err)
		}
		if !time.Time(z).IsZero() {
			t.Errorf("CustomTime(%s) = %v, ожидалось нулевое время", in, time.Time(z))
		}
	}
}

func TestClearString(t *testing.T) {
	cases := map[string]string{
		"w 71295":      "W71295",
		"KR9P-50-7TZB": "KR9P507TZB",
		"7L6_201/338":  "7L6201338",
		"№123":         "123",
	}
	for in, want := range cases {
		if got := ClearString(in); got != want {
			t.Errorf("ClearString(%q) = %q, ожидалось %q", in, got, want)
		}
	}
}

// TestMoney страхует показ баланса: накопленный float-остаток и минус ноль
// должны становиться обычным нулём, а не "-0.00" в цвете долга.
func TestMoney(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want float64
	}{
		{"остаток от сложения накладных", -1.4210854715202004e-14, 0},
		{"минус ноль", math.Copysign(0, -1), 0},
		{"ноль", 0, 0},
		{"долг", -123.456, -123.46},
		{"аванс", 123.454, 123.45},
		{"копейка долга остаётся долгом", -0.01, -0.01},
		{"меньше полкопейки схлопывается в ноль", -0.004, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Money(c.in)
			if got != c.want {
				t.Fatalf("Money(%v) = %v, ожидалось %v", c.in, got, c.want)
			}
			if got == 0 && math.Signbit(got) {
				t.Fatalf("Money(%v) вернула минус ноль", c.in)
			}
		})
	}
}
