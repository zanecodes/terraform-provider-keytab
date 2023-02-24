package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccFileResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccFileResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keytab_file.test", "content_base64", "BQIAAAA5AAEACXJlYWxtLmNvbQAJcHJpbmNpcGFsAAAAAQAAAAAAABcAEDHWz+DRaukxtzxZ1+DAicAAAAAA"),
					resource.TestCheckResourceAttrSet("keytab_file.test", "id"),
				),
			},
			// Update and Read testing
			{
				Config: testAccFileResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keytab_file.test", "content_base64", "BQIAAAA5AAEACXJlYWxtLmNvbQAJcHJpbmNpcGFsAAAAAQAAAAAAABcAEDHWz+DRaukxtzxZ1+DAicAAAAAA"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccFileResourceConfig() string {
	return fmt.Sprintf(`
resource "keytab_file" "test" {
  entry {
    principal = "principal"
    realm = "realm.com"
  }
}
`)
}
