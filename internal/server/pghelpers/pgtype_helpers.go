package pghelpers

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// PgtypeUUIDToUUID 將 pgtype.UUID 轉換為 uuid.UUID
func PgtypeUUIDToUUID(pgu pgtype.UUID) uuid.UUID {
	if !pgu.Valid {
		return uuid.Nil
	}
	return pgu.Bytes
}

// UUIDToPgtypeUUID 將 uuid.UUID 轉換為 pgtype.UUID
func UUIDToPgtypeUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: u,
		Valid: u != uuid.Nil,
	}
}

// PgtypeTextToStringPtr 將 pgtype.Text 轉換為 *string
func PgtypeTextToStringPtr(pt pgtype.Text) *string {
	if !pt.Valid {
		return nil
	}
	return &pt.String
}

// StringToPgtypeText 將 string 轉換為 pgtype.Text
func StringToPgtypeText(s string) pgtype.Text {
	return pgtype.Text{
		String: s,
		Valid:  true,
	}
}

// StringPtrToPgtypeText 將 *string 轉換為 pgtype.Text
func StringPtrToPgtypeText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{
		String: *s,
		Valid:  true,
	}
}

// PgtypeTimestamptzToTimePtr 將 pgtype.Timestamptz 轉換為 *time.Time
func PgtypeTimestamptzToTimePtr(pt pgtype.Timestamptz) *time.Time {
	if !pt.Valid {
		return nil
	}
	return &pt.Time
}

// TimeToPgtypeTimestamptz 將 time.Time 轉換為 pgtype.Timestamptz
func TimeToPgtypeTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  t,
		Valid: !t.IsZero(),
	}
}

// TimePtrToPgtypeTimestamptz 將 *time.Time 轉換為 pgtype.Timestamptz
func TimePtrToPgtypeTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{
		Time:  *t,
		Valid: true,
	}
}

// PgtypeInt4ToInt32 將 pgtype.Int4 轉換為 int32
func PgtypeInt4ToInt32(pi pgtype.Int4) int32 {
	if !pi.Valid {
		return 0
	}
	return pi.Int32
}

// Int32ToPgtypeInt4 將 int32 轉換為 pgtype.Int4
func Int32ToPgtypeInt4(i int32) pgtype.Int4 {
	return pgtype.Int4{
		Int32: i,
		Valid: true,
	}
}

// PgtypeInt4ToInt32Ptr 將 pgtype.Int4 轉換為 *int32
func PgtypeInt4ToInt32Ptr(pi pgtype.Int4) *int32 {
	if !pi.Valid {
		return nil
	}
	return &pi.Int32
}

// PgtypeNumericToFloat64 將 pgtype.Numeric 轉換為 float64
func PgtypeNumericToFloat64(pn pgtype.Numeric) float64 {
	if !pn.Valid {
		return 0
	}

	// 嘗試轉換為 float64
	f64, err := pn.Float64Value()
	if err != nil {
		return 0
	}

	return f64.Float64
}

// Float64ToPgtypeNumeric 將 float64 轉換為 pgtype.Numeric
func Float64ToPgtypeNumeric(f float64) pgtype.Numeric {
	var pn pgtype.Numeric
	if err := pn.Scan(f); err != nil {
		return pgtype.Numeric{Valid: false}
	}
	return pn
}

// PgtypeTimestamptzToTime 將 pgtype.Timestamptz 轉換為 time.Time
func PgtypeTimestamptzToTime(pt pgtype.Timestamptz) time.Time {
	if !pt.Valid {
		return time.Time{}
	}
	return pt.Time
}
