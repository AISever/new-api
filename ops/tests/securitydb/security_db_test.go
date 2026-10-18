package securitydb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSecurityMaintenanceDatabaseContracts(t *testing.T) {
	for index, dialect := range []string{"sqlite", "postgres", "mysql"} {
		t.Run(dialect, func(t *testing.T) {
			dsn := os.Getenv("SECURITY_TEST_" + strings.ToUpper(dialect) + "_DSN")
			if dialect == "sqlite" {
				common.SQLitePath = filepath.Join(t.TempDir(), "security.db")
				dsn = "local"
			} else if dsn == "" {
				t.Skip("set a dedicated SECURITY_TEST_" + strings.ToUpper(dialect) + "_DSN")
			} else if !strings.Contains(dsn, "security_test") {
				t.Fatal("refusing a database outside the security_test namespace")
			}
			t.Setenv("SQL_DSN", dsn)
			common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = false, false, false
			common.RedisEnabled, common.BatchUpdateEnabled, common.IsMasterNode = false, false, true
			require.NoError(t, model.InitDB())
			require.NoError(t, model.InitLogDB())
			t.Cleanup(func() { _ = model.CloseDB() })
			user := model.User{Username: fmt.Sprintf("sec-%d-%d", index, time.Now().UnixNano()), Password: "synthetic", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 1000, AffCode: common.GetRandomString(8)}
			require.NoError(t, model.DB.Create(&user).Error)
			stale, err := model.GetUserById(user.Id, true)
			require.NoError(t, err)
			require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", user.Id).Update("quota", gorm.Expr("quota - ?", 400)).Error)
			stale.DisplayName = "updated"
			require.NoError(t, stale.Update(false))
			require.NoError(t, model.UpdateUserSetting(user.Id, dto.UserSetting{BillingPreference: "subscription_first", Language: "en"}))
			var current model.User
			require.NoError(t, model.DB.First(&current, user.Id).Error)
			require.Equal(t, 600, current.Quota)
			require.Equal(t, "updated", current.DisplayName)
			plan := model.SubscriptionPlan{Id: 991310 + index, Title: "security midnight plan", DurationUnit: "month", DurationValue: 1, Enabled: true, TotalAmount: 10000, EffectiveGroups: `["allowed"]`, QuotaResetPeriod: model.SubscriptionResetCustom, QuotaResetCustomSeconds: 86400}
			// A repeat run updates this test-only plan without changing any production data.
			require.NoError(t, model.DB.Save(&plan).Error)
			now := time.Now()
			zone, err := time.LoadLocation("Asia/Shanghai")
			require.NoError(t, err)
			local := now.In(zone)
			midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone).Unix()
			sub := model.UserSubscription{UserId: user.Id, PlanId: plan.Id, AmountTotal: 10000, AmountUsed: 99, StartTime: midnight - 172800, EndTime: midnight + 172800, Status: "active", LastResetTime: midnight - 86400, NextResetTime: midnight}
			require.NoError(t, model.DB.Create(&sub).Error)
			_, err = model.PreConsumeUserSubscription(fmt.Sprintf("sec-allowed-%d-%d", index, now.UnixNano()), user.Id, "allowed", "security-model", 0, 50)
			require.NoError(t, err)
			require.NoError(t, model.DB.First(&sub, sub.Id).Error)
			require.EqualValues(t, 50, sub.AmountUsed, "due midnight resets before a new charge")
			require.Equal(t, midnight, sub.LastResetTime)
			require.Equal(t, midnight+86400, sub.NextResetTime)
			_, err = model.PreConsumeUserSubscription(fmt.Sprintf("sec-denied-%d-%d", index, now.UnixNano()), user.Id, "denied", "security-model", 0, 10)
			require.Error(t, err, "private effective-group restriction must remain enforced")
			require.NoError(t, model.DB.First(&sub, sub.Id).Error)
			require.EqualValues(t, 50, sub.AmountUsed)
			require.NoError(t, model.PostConsumeUserSubscriptionDelta(sub.Id, 10))
			require.NoError(t, model.DB.First(&sub, sub.Id).Error)
			require.EqualValues(t, 60, sub.AmountUsed)
		})
	}
}
