package utilits

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// clearReplacer строится один раз: strings.NewReplacer собирает внутреннее
// дерево поиска, и пересоздавать его на каждый вызов ClearString незачем.
var clearReplacer = newClearReplacer()

func newClearReplacer() *strings.Replacer {
	return strings.NewReplacer(
		"~", "",
		"`", "",
		"!", "",
		"@", "",
		"#", "",
		"$", "",
		"%", "",
		"^", "",
		"&", "",
		"*", "",
		"(", "",
		")", "",
		"_", "",
		"+", "",
		"-", "",
		"=", "",
		"{", "",
		"}", "",
		"[", "",
		"]", "",
		",", "",
		"/", "",
		"?", "",
		":", "",
		"<", "",
		">", "",
		"'", "",
		";", "",
		`\`, "",
		`"`, "",
		"|", "",
		"№", "",
		" ", "",
	)
}

func ClearString(s string) string {
	return strings.ToUpper(clearReplacer.Replace(s))
}

func BoolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

type (
	CustomFloat64 struct {
		Float64 float64
	}
	CustomInt64 struct {
		Int64 int64
	}
	CustomInt struct {
		Int int
	}
	CustomBool bool
	CustomTime time.Time
)

const QUOTES_BYTE = 34

func (cb *CustomBool) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case `"TRUE"`, `TRUE`, `"true"`, `true`, `"1"`, `1`:
		*cb = true
	case `"FALSE"`, `FALSE`, `"false"`, `false`, `"0"`, `0`, `""`:
		*cb = false
	default:
		return fmt.Errorf(`CustomBool: parsing "%s": unknown value`, string(data))
	}
	return nil
}

// Числовые поля Техномир приходят то числом, то строкой, то прочерком, поэтому
// разбираются вручную. Раньше внутри вызывался json.Unmarshal — на каждое поле
// каждого предложения это отдельный проход декодера; в выдаче по «широкому»
// артикулу таких полей десятки тысяч. strconv делает то же самое напрямую.
// Пустое значение, прочерк и null дают ноль (json.Unmarshal на null тоже
// оставлял ноль — сохраняем это поведение).
func isEmptyNumber(s string) bool {
	return s == "" || s == "-" || s == "null"
}

func (cf *CustomFloat64) UnmarshalJSON(data []byte) error {
	if data[0] == QUOTES_BYTE {
		// Разделитель тысяч: "1,234.56" → "1234.56".
		raw := string(bytes.Replace(data[1:len(data)-1], []byte{44}, []byte{}, 1))
		if isEmptyNumber(raw) {
			cf.Float64 = 0
			return nil
		}
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("CustomFloat64: UnmarshalJSON with quotes data [%s]: %w", raw, err)
		}
		cf.Float64 = val
		return nil
	}
	raw := string(data)
	if isEmptyNumber(raw) {
		cf.Float64 = 0
		return nil
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fmt.Errorf("CustomFloat64: UnmarshalJSON without quotes data [%s]: %w", raw, err)
	}
	cf.Float64 = val
	return nil
}

func (ci *CustomInt64) UnmarshalJSON(data []byte) error {
	raw := string(data)
	if data[0] == QUOTES_BYTE {
		raw = string(data[1 : len(data)-1])
	}
	if isEmptyNumber(raw) {
		ci.Int64 = 0
		return nil
	}
	val, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("CustomInt64: UnmarshalJSON: %w", err)
	}
	ci.Int64 = val
	return nil
}

func (ci *CustomInt) UnmarshalJSON(data []byte) error {
	raw := string(data)
	if data[0] == QUOTES_BYTE {
		raw = string(data[1 : len(data)-1])
	}
	if isEmptyNumber(raw) {
		ci.Int = 0
		return nil
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("CustomInt: UnmarshalJSON: %w", err)
	}
	ci.Int = val
	return nil
}

func (ct *CustomTime) UnmarshalJSON(data []byte) error {
	if data[0] == QUOTES_BYTE {
		if string(data[1:len(data)-1]) == "-" {
			*ct = CustomTime{}
			return nil
		}

		t, err := time.Parse("2006-01-02 15:04:05", string(data[1:len(data)-1]))
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05.000000", string(data[1:len(data)-1]))
			if err != nil {
				*ct = CustomTime{}
				return nil
			} else {
				*ct = CustomTime(t)
				return nil
			}
		}
		*ct = CustomTime(t)
		return nil
	}
	return fmt.Errorf(`CustomTime: parsing "%s": unknown value`, string(data))
}

func Cut(text string, limit int) string {
	runes := []rune(text)
	if len(runes) >= limit {
		return string(runes[:limit]) + "..."
	}
	return text
}

// Money приводит денежную сумму к копейкам: округляет до двух знаков и убирает
// "минус ноль". Балансы складываются из накладных в float64, поэтому взаимно
// погашенные суммы дают остаток вида -1.4e-14 — он печатался как -0.00 и красился
// в цвет долга. RoundWithPrecission тут не годится: для отрицательных значений
// она округляет через Floor (-1.4e-14 превратилось бы в -0.01).
func Money(x float64) float64 {
	rounded := math.Round(x*100) / 100
	if rounded == 0 {
		// Явный +0: у float64 минус ноль сохраняется и печатается как "-0.00".
		return 0
	}
	return rounded
}

func RoundWithPrecission(x float64, prec int) float64 {
	var rounder float64
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	_, frac := math.Modf(intermed)
	if frac >= 0.5 {
		rounder = math.Ceil(intermed)
	} else {
		rounder = math.Floor(intermed)
	}
	return rounder / pow
}

func MustParseUUID(u string) uuid.UUID {
	defer func() {
		if recover() != nil {
			var nu uuid.NullUUID
			uuid.MustParse(nu.UUID.String())
		}
	}()
	return uuid.MustParse(u)
}

func IsNullOrWrongUUID(u uuid.UUID) bool {
	var null uuid.NullUUID
	return u == null.UUID
}

// func ReverseSlice(s interface{}) {
// 	size := reflect.ValueOf(s).Len()
// 	swap := reflect.Swapper(s)
// 	for i, j := 0, size-1; i < j; i, j = i+1, j-1 {
// 		swap(i, j)
// 	}
// }

// func reverse(numbers []int) []int {
// 	newNumbers := make([]int, 0, len(numbers))
// 	for i := len(numbers)-1; i >= 0; i-- {
// 		newNumbers = append(newNumbers, numbers[i])
// 	}
// 	return newNumbers
// }

func DecodeMonth(monthRu string) time.Month {
	switch monthRu {
	case "Январь":
		return time.January
	case "Февраль":
		return time.February
	case "Март":
		return time.March
	case "Апрель":
		return time.April
	case "Май":
		return time.May
	case "Июнь":
		return time.June
	case "Июль":
		return time.July
	case "Август":
		return time.August
	case "Сентябрь":
		return time.September
	case "Октябрь":
		return time.October
	case "Ноябрь":
		return time.November
	case "Декабрь":
		return time.December
	}
	return -1
}

// parseRuDate парсит дату из формата виджета календаря "Месяц D, YYYY"
// (например "Август 10, 2026"), с фолбэком на ISO "2006-01-02" на случай,
// если поле не было тронуто пользователем и содержит исходное значение,
// отрендеренное сервером.
func parseRuDate(date string) (time.Time, bool) {
	parts := strings.Split(date, " ")
	if len(parts) < 3 {
		if t, err := time.Parse("2006-01-02", date); err == nil {
			return t, true
		}
		return time.Time{}, false
	}
	flag := "02"
	if len(parts[1]) == 2 {
		flag = "2"
	}
	t, err := time.Parse(fmt.Sprintf("1 %s, 2006", flag), fmt.Sprintf("%d %s %s", DecodeMonth(parts[0]), parts[1], parts[2]))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func RangeTimesFromStrings(start, end string) (time.Time, time.Time) {
	tme, ok := parseRuDate(end)
	if !ok {
		tme = time.Now()
	} else {
		// Конец периода приходит без времени (полночь), из-за чего заказы,
		// созданные позже полуночи в этот же день, не попадали в выборку.
		tme = time.Date(tme.Year(), tme.Month(), tme.Day(), 23, 59, 59, 999999999, tme.Location())
	}
	tms, ok := parseRuDate(start)
	if !ok {
		tms = tme.Add(-720 * time.Hour)
	}
	return tms, tme
}

// func StatusContainsStatuses(status int, statuses []domain.StatusDetail) bool {
// 	for _, s := range statuses {
// 		if status == int(s) {
// 			return true
// 		}
// 	}
// 	return false
// }
