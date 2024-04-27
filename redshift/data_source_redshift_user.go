package redshift

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceRedshiftUser() *schema.Resource {
	return &schema.Resource{
		Description: `
This data source can be used to fetch information about a specific database user. Users are authenticated when they login to Amazon Redshift. They can own databases and database objects (for example, tables) and can grant privileges on those objects to users, groups, and schemas to control who has access to which object. Users with CREATE DATABASE rights can create databases and grant privileges to those databases. Superusers have database ownership privileges for all databases.
`,
		ReadWithoutTimeout: RedshiftResourceFunc(dataSourceRedshiftUserRead),
		Schema: map[string]*schema.Schema{
			userNameAttr: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the user account. The user name can't be `PUBLIC`.",
				ValidateFunc: validation.StringNotInSlice([]string{
					"public",
				}, true),
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
			},
			userValidUntilAttr: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date and time after which the user's password is no longer valid. By default the password has no time limit.",
			},
			userCreateDBAttr: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates whether the user is allowed to create new databases.",
			},
			userConnLimitAttr: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The maximum number of database connections the user is permitted to have open concurrently. The limit isn't enforced for superusers.",
			},
			userSyslogAccessAttr: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A clause that specifies the level of access that the user has to the Amazon Redshift system tables and views. If `RESTRICTED` (default) is specified, the user can see only the rows generated by that user in user-visible system tables and views. If `UNRESTRICTED` is specified, the user can see all rows in user-visible system tables and views, including rows generated by another user. `UNRESTRICTED` doesn't give a regular user access to superuser-visible tables. Only superusers can see superuser-visible tables.",
			},
			userSuperuserAttr: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: `Indicates whether the user is a superuser with all database privileges.`,
			},
			userSessionTimeoutAttr: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The maximum time in seconds that a session remains inactive or idle. The range is 60 seconds (one minute) to 1,728,000 seconds (20 days). If no session timeout is set for the user, the cluster setting applies.",
			},
		},
	}
}

func dataSourceRedshiftUserRead(db *DBConnection, d *schema.ResourceData) error {
	var useSysID, userValidUntil, userConnLimit, userSyslogAccess, userSessionTimeout string
	var userSuperuser, userCreateDB bool

	columns := []string{
		"user_id",
		"createdb",
		"superuser",
		"syslog_access",
		`COALESCE(connection_limit::TEXT, 'UNLIMITED')`,
		"session_timeout",
	}

	values := []interface{}{
		&useSysID,
		&userCreateDB,
		&userSuperuser,
		&userSyslogAccess,
		&userConnLimit,
		&userSessionTimeout,
	}

	userName := d.Get(userNameAttr).(string)

	userSQL := fmt.Sprintf("SELECT %s FROM svv_user_info WHERE user_name = $1", strings.Join(columns, ","))
	err := db.QueryRow(userSQL, userName).Scan(values...)
	if err != nil {
		return err
	}

	err = db.QueryRow("SELECT COALESCE(valuntil, 'infinity') FROM pg_user_info WHERE usesysid = $1", useSysID).Scan(&userValidUntil)
	if err != nil {
		return err
	}

	userConnLimitNumber := -1
	if userConnLimit != "UNLIMITED" {
		if userConnLimitNumber, err = strconv.Atoi(userConnLimit); err != nil {
			return err
		}
	}

	userSessionTimeoutNumber, err := strconv.Atoi(userSessionTimeout)
	if err != nil {
		return err
	}

	d.SetId(useSysID)
	d.Set(userCreateDBAttr, userCreateDB)
	d.Set(userSuperuserAttr, userSuperuser)
	d.Set(userSyslogAccessAttr, userSyslogAccess)
	d.Set(userConnLimitAttr, userConnLimitNumber)
	d.Set(userValidUntilAttr, userValidUntil)
	d.Set(userSessionTimeoutAttr, userSessionTimeoutNumber)

	return nil
}
