package model

import (
	"errors"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm/clause"
)

const InvitationVisibilityOption = "InvitationAmountVisibility"

type InvitationVisibility struct {
	Mode    string `json:"mode"`
	UserIDs []int  `json:"user_ids"`
}
type InvitationUser struct {
	ID       int                `json:"-" gorm:"column:id"`
	Username string             `json:"username"`
	Amounts  *InvitationAmounts `json:"amounts,omitempty" gorm:"-"`
}
type InvitationConversion struct {
	QuotaPerUnit string `json:"quota_per_unit"`
	ExchangeRate string `json:"exchange_rate"`
	Estimated    bool   `json:"estimated"`
}
type InvitationCurrencyAmount struct {
	Conversions          []InvitationConversion `json:"-"`
	ConversionIncomplete bool                   `json:"-"`
	Currency             string                 `json:"currency"`
	CurrencySymbol       string                 `json:"currency_symbol,omitempty"`
	AdminTopUp           string                 `json:"-"`
	TopUp                string                 `json:"-"`
	Subscription         string                 `json:"-"`
	Redemption           string                 `json:"-"`
	Total                string                 `json:"total"`
	Estimated            bool                   `json:"estimated"`
}
type InvitationAmounts struct {
	Currencies      []InvitationCurrencyAmount `json:"currencies"`
	UnknownCount    int                        `json:"unknown_count"`
	UnpricedCount   int                        `json:"unpriced_count"`
	LegacyTruncated bool                       `json:"-"`
	LegacyRecorded  []InvitationLegacyAmount   `json:"-"`
}
type InvitationPage struct {
	Items          []InvitationUser `json:"items"`
	Total          int64            `json:"total"`
	Page           int              `json:"page"`
	PageSize       int              `json:"page_size"`
	CanViewAmounts bool             `json:"can_view_amounts"`
}

func parseInvitationVisibility(raw string) (InvitationVisibility, error) {
	var p InvitationVisibility
	if err := common.UnmarshalJsonStr(raw, &p); err != nil {
		return p, errors.New("邀请金额权限配置无效")
	}
	if p.Mode != "off" && p.Mode != "all" && p.Mode != "selected" {
		return p, errors.New("邀请金额可见范围无效")
	}
	if len(p.UserIDs) > 1000 {
		return p, errors.New("指定用户不能超过1000人")
	}
	seen := map[int]bool{}
	ids := make([]int, 0, len(p.UserIDs))
	for _, id := range p.UserIDs {
		if id <= 0 {
			return p, errors.New("用户ID必须是正整数")
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	p.UserIDs = ids
	return p, nil
}
func ValidateInvitationVisibility(raw string) (string, error) {
	p, err := parseInvitationVisibility(raw)
	if err != nil {
		return "", err
	}
	if p.Mode == "selected" && len(p.UserIDs) > 0 {
		var count int64
		if err = DB.Model(&User{}).Where("id IN ?", p.UserIDs).Count(&count).Error; err != nil {
			return "", err
		}
		if count != int64(len(p.UserIDs)) {
			return "", errors.New("指定用户不存在或已删除")
		}
	}
	b, err := common.Marshal(p)
	return string(b), err
}
func saveInvitationVisibility(raw string) error {
	value, err := ValidateInvitationVisibility(raw)
	if err != nil {
		return err
	}
	option := Option{Key: InvitationVisibilityOption, Value: value}
	if err = DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoUpdates: clause.AssignmentColumns([]string{"value"})}).Create(&option).Error; err != nil {
		return err
	}
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap[InvitationVisibilityOption] = value
	return nil
}

func InvitationVisibilityRaw() (string, error) {
	var option Option
	result := DB.Where(&Option{Key: InvitationVisibilityOption}).Limit(1).Find(&option)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 {
		return `{"mode":"off","user_ids":[]}`, nil
	}
	return option.Value, nil
}

// Read the persisted row on every request: revocations apply across application instances.
func CanViewInvitationAmounts(userID int) bool {
	if userID <= 0 {
		return false
	}
	var option Option
	if DB.Where(&Option{Key: InvitationVisibilityOption}).Take(&option).Error != nil {
		return false
	}
	p, err := parseInvitationVisibility(option.Value)
	if err != nil {
		return false
	}
	if p.Mode == "all" {
		return activeInvitationViewer(userID)
	}
	if p.Mode == "selected" {
		for _, id := range p.UserIDs {
			if id == userID {
				return activeInvitationViewer(userID)
			}
		}
	}
	return false
}
func InvitationRootActive(id int) bool {
	if id <= 0 {
		return false
	}
	var count int64
	return DB.Model(&User{}).Where(&User{Id: id, Role: common.RoleRootUser, Status: common.UserStatusEnabled}).Count(&count).Error == nil && count == 1
}

func activeInvitationViewer(id int) bool {
	var count int64
	return DB.Model(&User{}).Where("id = ? AND status = ?", id, common.UserStatusEnabled).Count(&count).Error == nil && count == 1
}

func invitationPattern(keyword string) string {
	return "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(keyword) + "%"
}
func GetInvitationPage(inviterID, page, pageSize int, keyword string, withAmounts bool) (*InvitationPage, error) {
	if inviterID <= 0 || page < 1 || page > 1000000 || pageSize < 1 || pageSize > 100 || len(keyword) > 100 {
		return nil, errors.New("邀请名单分页或搜索参数无效")
	}
	result := &InvitationPage{Items: []InvitationUser{}, Page: page, PageSize: pageSize, CanViewAmounts: withAmounts}
	query := DB.Model(&User{}).Where("inviter_id = ?", inviterID)
	if keyword != "" {
		query = query.Where("username LIKE ? ESCAPE '!'", invitationPattern(keyword))
	}
	if err := query.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	if err := query.Select("id", "username").Order("id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&result.Items).Error; err != nil {
		return nil, err
	}
	if withAmounts {
		if err := AttachInvitationAmounts(result.Items); err != nil {
			return nil, err
		}
	}
	return result, nil
}

type InvitationUserChoice struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

func GetInvitationUserChoices(keyword string, ids []int) ([]InvitationUserChoice, error) {
	if len(keyword) > 100 || len(ids) > 100 {
		return nil, errors.New("用户搜索参数无效")
	}
	result := []InvitationUserChoice{}
	q := DB.Model(&User{})
	if len(ids) > 0 {
		q = q.Where("id IN ?", ids)
	} else if keyword != "" {
		q = q.Where("username LIKE ? ESCAPE '!'", invitationPattern(keyword))
	}
	err := q.Select("id", "username").Order("id DESC").Limit(100).Find(&result).Error
	return result, err
}
