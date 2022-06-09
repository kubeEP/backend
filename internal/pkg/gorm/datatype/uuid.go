package gormDatatype

import (
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type UUID uuid.UUID

func (UUID) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return "UUID"
}

func (UUID) GormDataType() string {
	return "uuid"
}

func (u UUID) GetUUID() uuid.UUID {
	return uuid.UUID(u)
}

func (u *UUID) SetUUID(id uuid.UUID) {
	*u = UUID(id)
}

func (u *UUID) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil

	case string:
		// if an empty UUID comes from a table, we return a null UUID
		if src == "" {
			return nil
		}

		// see Parse for required string format
		uid, err := uuid.Parse(src)
		if err != nil {
			return fmt.Errorf("Scan: %v", err)
		}

		*u = UUID(uid)

	case []byte:
		// if an empty UUID comes from a table, we return a null UUID
		if len(src) == 0 {
			return nil
		}

		// assumes a simple slice of bytes if 16 bytes
		// otherwise attempts to parse
		if len(src) != 16 {
			return u.Scan(string(src))
		}
		copy((*u)[:], src)

	default:
		return fmt.Errorf("Scan: unable to scan type %T into UUID", src)
	}

	return nil
}
