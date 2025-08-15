package config

import "testing"

func TestBind(t *testing.T) {
	type Settings struct {
		ID string `config:"id" check:"required,uuid"`
	}

	type testStruct struct {
		Port     int      `config:"app.port" check:"default=8080,min=1000,max=9000"`
		Email    string   `config:"email" check:"required,email"`
		Username string   `config:"username" check:"required,match='^[A-Za-z0-9_]+$'"`
		Settings Settings `config:"settings"`
	}

	c := New()
	// simulate loaded config
	c.Set("app.port", 1500)
	c.Set("email", "user@example.com")
	c.Set("username", "valid_user")
	c.Set("settings.id", "550e8400-e29b-41d4-a716-446655440000")

	var ts testStruct
	if err := c.Bind(&ts); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}

	if ts.Port != 1500 {
		t.Errorf("expected Port=1500, got %d", ts.Port)
	}
	if ts.Email != "user@example.com" {
		t.Errorf("expected Email=user@example.com, got %s", ts.Email)
	}
	if ts.Username != "valid_user" {
		t.Errorf("expected Username=valid_user, got %s", ts.Username)
	}
	if ts.Settings.ID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected ID to match uuid, got %s", ts.Settings.ID)
	}
}

func TestBindDefaultsAndValidationFail(t *testing.T) {
	type testStruct struct {
		Port  int    `config:"app.port" check:"default=8080,min=1000,max=9000"`
		Email string `config:"email" check:"required,email"`
	}

	c := New()
	// no values set, should use default for port, fail for email
	var ts testStruct
	err := c.Bind(&ts)
	if err == nil {
		t.Fatal("expected validation error for missing required email, got nil")
	}
	if ts.Port != 8080 {
		t.Errorf("expected default Port=8080, got %d", ts.Port)
	}
}
