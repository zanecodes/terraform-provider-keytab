package provider

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/jcmturner/gokrb5/v8/iana/etypeID"
	"github.com/jcmturner/gokrb5/v8/keytab"
)

func TestAccFileResource(t *testing.T) {
	first_keytab := keytab.New()

	second_keytab := keytab.New()
	if err := second_keytab.AddEntry("principal", "realm.com", "key", time.Unix(0, 0), 0, etypeID.RC4_HMAC); err != nil {
		t.Fatal(err.Error())
		return
	}

	third_keytab := keytab.New()
	if err := third_keytab.AddEntry("principal", "realm.com", "key", time.Unix(0, 0), 0, etypeID.RC4_HMAC); err != nil {
		t.Fatal(err.Error())
		return
	}
	if err := third_keytab.AddEntry("principal two", "realm-two.com", "key two", time.Unix(1, 0), 1, etypeID.AES128_CTS_HMAC_SHA1_96); err != nil {
		t.Fatal(err.Error())
		return
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: `
resource "keytab_file" "test" {
  entry {
    principal = "principal"
    realm = "realm.com"
    key = "key"
    key_version = 0
    encryption_type = "rc4-hmac"
  }
}
`,
				PreConfig: func() {
					if err := first_keytab.AddEntry("principal", "realm.com", "key", time.Now().Truncate(time.Second), 0, etypeID.RC4_HMAC); err != nil {
						t.Fatal(err.Error())
						return
					}
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keytab_file.test", "id"),
					resource.TestCheckResourceAttrWith("keytab_file.test", "content_base64", testAccCheckKeytabContent(t, first_keytab)),
				),
			},
			// Update and Read testing
			{
				Config: `
resource "keytab_file" "test" {
  entry {
    principal = "principal"
    realm = "realm.com"
    key = "key"
    key_version = 0
    encryption_type = "rc4-hmac"
    timestamp = "1970-01-01T00:00:00Z"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("keytab_file.test", "content_base64", testAccCheckKeytabContent(t, second_keytab)),
				),
			},
			{
				Config: `
resource "keytab_file" "test" {
  entry {
    principal = "principal"
    realm = "realm.com"
    key = "key"
    key_version = 0
    encryption_type = "rc4-hmac"
    timestamp = "1970-01-01T00:00:00Z"
  }
  entry {
    principal = "principal two"
    realm = "realm-two.com"
    key = "key two"
    key_version = 1
    encryption_type = "aes128-sha1"
    timestamp = "1970-01-01T00:00:01Z"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("keytab_file.test", "content_base64", testAccCheckKeytabContent(t, third_keytab)),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccCheckKeytabContent(_ *testing.T, expected *keytab.Keytab) resource.CheckResourceAttrWithFunc {
	return func(actual_value string) error {
		expected_bytes, err := expected.Marshal()

		if err != nil {
			return err
		}

		expected_value := base64.StdEncoding.EncodeToString(expected_bytes)

		if actual_value != expected_value {
			actual_bytes, err := base64.StdEncoding.DecodeString(actual_value)

			if err != nil {
				return err
			}

			actual := keytab.New()
			err = actual.Unmarshal(actual_bytes)

			if err != nil {
				return err
			}

			expected_json, err := expected.JSON()

			if err != nil {
				return err
			}

			actual_json, err := actual.JSON()

			if err != nil {
				return err
			}

			return fmt.Errorf("Expected keytab:\n%s\nActual keytab:\n%s", expected_json, actual_json)
		}

		return nil
	}
}
