package tests_test

import (
	"reflect"
	"testing"

	"gorm.io/gorm"
)

type SetColumnPartialModel struct {
	ID        uint
	PriceArea string
	Address   string
}

func (m *SetColumnPartialModel) BeforeUpdate(tx *gorm.DB) error {
	tx.Statement.SetColumn("price_area", "hook")
	return nil
}

// A struct update only assigns columns represented by the update schema.
// SetColumn must not overwrite a different field or reject a partial update.
func TestSetColumnMissingUpdateField(t *testing.T) {
	if err := DB.AutoMigrate(&SetColumnPartialModel{}); err != nil {
		t.Fatal(err)
	}
	m := SetColumnPartialModel{PriceArea: "original", Address: "original"}
	if err := DB.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	payload := struct {
		ID      uint
		Address string
		Other   string
	}{Address: "new", Other: "keep"}
	if err := DB.Model(&m).Updates(&payload).Error; err != nil {
		t.Fatal(err)
	}
	var got SetColumnPartialModel
	if err := DB.First(&got, m.ID).Error; err != nil {
		t.Fatal(err)
	}
	storedControl := setColumnPartialControl(t, &payload)
	if got.Address != storedControl.Address || got.PriceArea != storedControl.PriceArea {
		t.Fatalf("hook changed an absent DTO column: hooked=%+v control=%+v", got, storedControl)
	}
	if payload != (struct {
		ID      uint
		Address string
		Other   string
	}{Address: "new", Other: "keep"}) || m.PriceArea != "hook" {
		t.Fatalf("unexpected partial update: stored=%+v payload=%+v model=%+v", got, payload, m)
	}
}

func setColumnPartialControl(t *testing.T, payload interface{}) SetColumnPartialModel {
	t.Helper()
	// The existing assignment builder omits absent DTO columns.
	control := SetColumnPartialModel{PriceArea: "original", Address: "original"}
	if err := DB.Create(&control).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Session(&gorm.Session{SkipHooks: true}).Model(&control).Updates(payload).Error; err != nil {
		t.Fatal(err)
	}
	var storedControl SetColumnPartialModel
	if err := DB.First(&storedControl, control.ID).Error; err != nil {
		t.Fatal(err)
	}
	return storedControl
}

type SetColumnColumnModel struct {
	ID      uint
	Target  string `gorm:"column:Value"`
	Address string
}

func (m *SetColumnColumnModel) BeforeUpdate(tx *gorm.DB) error {
	tx.Statement.SetColumn("Target", "hook")
	return nil
}

func TestSetColumnMatchesDatabaseColumn(t *testing.T) {
	if err := DB.AutoMigrate(&SetColumnColumnModel{}); err != nil {
		t.Fatal(err)
	}
	for _, renamed := range []bool{false, true} {
		t.Run(map[bool]string{false: "same_name_other_column", true: "renamed_same_column"}[renamed], func(t *testing.T) {
			checkSetColumnDatabaseColumn(t, renamed)
		})
	}
}

func checkSetColumnDatabaseColumn(t *testing.T, renamed bool) {
	t.Helper()
	m := SetColumnColumnModel{Target: "original", Address: "original"}
	if err := DB.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	if renamed {
		updateSetColumnRenamed(t, &m)
	} else {
		updateSetColumnUnrelated(t, &m)
	}
	var got SetColumnColumnModel
	if err := DB.First(&got, m.ID).Error; err != nil {
		t.Fatal(err)
	}
	want := "original"
	if renamed {
		want = "hook"
	}
	if got.Target != want || got.Address != "new" {
		t.Fatalf("wrong columns: %+v", got)
	}
}

func updateSetColumnRenamed(t *testing.T, m *SetColumnColumnModel) {
	t.Helper()
	payload := struct {
		Address string
		Zone    string `gorm:"column:Value"`
	}{Address: "new", Zone: "requested"}
	if err := DB.Model(m).Updates(&payload).Error; err != nil {
		t.Fatal(err)
	}
	if payload.Zone != "hook" || payload.Address != "new" {
		t.Fatalf("wrong renamed payload: %+v", payload)
	}
}

func updateSetColumnUnrelated(t *testing.T, m *SetColumnColumnModel) {
	t.Helper()
	// Same Go name as either the model field or its DB column is not a column match.
	payload := struct {
		Target string `gorm:"column:other"`
		Value  string `gorm:"column:address"`
	}{Target: "keep", Value: "new"}
	if err := DB.Model(m).Updates(&payload).Error; err != nil {
		t.Fatal(err)
	}
	if payload.Target != "keep" || payload.Value != "new" {
		t.Fatalf("wrong unrelated payload: %+v", payload)
	}
}

type SetColumnSerializerModel struct {
	ID      uint
	Address string
	Tags    []string `gorm:"serializer:json"`
}

func (m *SetColumnSerializerModel) BeforeUpdate(tx *gorm.DB) error {
	tx.Statement.SetColumn("tags", []string{"hook", "value"})
	return nil
}

func TestSetColumnUpdateSerializer(t *testing.T) {
	if err := DB.AutoMigrate(&SetColumnSerializerModel{}); err != nil {
		t.Fatal(err)
	}
	m := SetColumnSerializerModel{Address: "original", Tags: []string{"original"}}
	if err := DB.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	payload := struct {
		Tags    []string `gorm:"serializer:json"`
		Address string
	}{Tags: []string{"requested"}, Address: "new"}
	if err := DB.Model(&m).Updates(&payload).Error; err != nil {
		t.Fatal(err)
	}
	var got SetColumnSerializerModel
	if err := DB.First(&got, m.ID).Error; err != nil {
		t.Fatal(err)
	}
	want := []string{"hook", "value"}
	if !reflect.DeepEqual(got.Tags, want) || !reflect.DeepEqual(payload.Tags, want) || got.Address != "new" || payload.Address != "new" {
		t.Fatalf("wrong serializer update: stored=%+v payload=%+v", got, payload)
	}
}
