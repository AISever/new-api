package model

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Explicit local-only opt-in; never accept a production address/database by mistake.
func TestInvitationAmountsExternalDatabase(t *testing.T) {
	driver, dsn := os.Getenv("INVITATION_TEST_DRIVER"), os.Getenv("INVITATION_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated external database not requested")
	}
	var dialector gorm.Dialector
	switch driver {
	case "postgres":
		parsed, err := url.Parse(dsn)
		require.NoError(t, err)
		require.Contains(t, []string{"127.0.0.1", "localhost"}, parsed.Hostname())
		require.True(t, strings.HasPrefix(strings.TrimPrefix(parsed.Path, "/"), "invitation_test"))
		dialector = postgres.Open(dsn)
	case "mysql":
		require.True(t, strings.Contains(dsn, "@tcp(127.0.0.1:") && strings.Contains(dsn, ")/invitation_test"), "only local invitation_test databases are allowed")
		dialector = mysql.Open(dsn)
	default:
		t.Fatal("unknown isolated test driver")
	}
	oldDB, oldLog := DB, LOG_DB
	oldSQLite, oldPostgres, oldMySQL := common.UsingSQLite, common.UsingPostgreSQL, common.UsingMySQL
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	DB = db
	LOG_DB = db
	common.UsingSQLite = false
	common.UsingPostgreSQL = driver == "postgres"
	common.UsingMySQL = driver == "mysql"
	t.Cleanup(func() {
		raw, _ := db.DB()
		_ = raw.Close()
		DB = oldDB
		LOG_DB = oldLog
		common.UsingSQLite = oldSQLite
		common.UsingPostgreSQL = oldPostgres
		common.UsingMySQL = oldMySQL
	})
	require.NoError(t, db.AutoMigrate(&User{}, &Option{}, &TopUp{}, &SubscriptionOrder{}, &Redemption{}, &UserSubscription{}, &InvitationAmountSnapshot{}))
	// A second migration must preserve the same schema/data and unique snapshot key.
	require.NoError(t, db.AutoMigrate(&InvitationAmountSnapshot{}))
	require.NoError(t, db.Create(&User{Id: 1, Username: "owner", AffCode: "db-owner", Quota: 9876}).Error)
	require.NoError(t, db.Create(&User{Id: 2, Username: "friend", AffCode: "db-friend", InviterId: 1}).Error)
	require.NoError(t, db.Create(&TopUp{Id: 1, UserId: 2, Money: 99, TradeNo: "external-paid", PaymentProvider: PaymentProviderStripe, Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, RecordInvitationPayment("external-paid", PaymentProviderStripe, "2.10", "USD"))
	require.NoError(t, RecordInvitationPayment("external-paid", PaymentProviderStripe, "2.10", "USD"))
	require.NoError(t, db.Create(&SubscriptionOrder{Id: 1, UserId: 2, Money: 7, TradeNo: "external-sub", PaymentProvider: PaymentProviderStripe, Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, db.Create(&TopUp{Id: 2, UserId: 2, Money: 7, TradeNo: "external-sub", Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, RecordInvitationPayment("external-sub", PaymentProviderStripe, "3.20", "USD"))
	require.NoError(t, UpdateOption(InvitationVisibilityOption, `{"mode":"selected","user_ids":[1]}`))
	require.True(t, CanViewInvitationAmounts(1))
	require.False(t, CanViewInvitationAmounts(2))
	page, err := GetInvitationPage(1, 1, 20, "", true)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "5.3", page.Items[0].Amounts.Currencies[0].Total)
	var user User
	require.NoError(t, db.First(&user, 1).Error)
	require.Equal(t, 9876, user.Quota)
}
