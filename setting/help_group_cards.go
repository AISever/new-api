package setting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const DefaultHelpGroupCardImageURL = "/WeChat-qun.jpg"

type HelpGroupCard struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	Label       string `json:"label"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	SortOrder   int    `json:"sort_order"`
}

var defaultHelpGroupCards = []HelpGroupCard{
	{
		ID:          "wechat-group-default",
		Enabled:     true,
		Label:       "群聊：AIGC交流二群",
		Title:       "欢迎加微信群沟通交流",
		Description: "扫描二维码加入 AIGC 交流群，与更多开发者一起探讨 AI 编程工具的使用技巧和最佳实践。",
		ImageURL:    DefaultHelpGroupCardImageURL,
		SortOrder:   0,
	},
}

var HelpGroupCards = cloneHelpGroupCards(defaultHelpGroupCards)

func cloneHelpGroupCards(cards []HelpGroupCard) []HelpGroupCard {
	if len(cards) == 0 {
		return nil
	}
	cloned := make([]HelpGroupCard, len(cards))
	copy(cloned, cards)
	return cloned
}

func DefaultHelpGroupCards() []HelpGroupCard {
	return cloneHelpGroupCards(defaultHelpGroupCards)
}

func sanitizeHelpGroupCards(cards []HelpGroupCard) []HelpGroupCard {
	sanitized := cloneHelpGroupCards(cards)
	sort.SliceStable(sanitized, func(i, j int) bool {
		if sanitized[i].SortOrder == sanitized[j].SortOrder {
			return sanitized[i].ID < sanitized[j].ID
		}
		return sanitized[i].SortOrder < sanitized[j].SortOrder
	})
	for i := range sanitized {
		sanitized[i].ID = strings.TrimSpace(sanitized[i].ID)
		sanitized[i].Label = strings.TrimSpace(sanitized[i].Label)
		sanitized[i].Title = strings.TrimSpace(sanitized[i].Title)
		sanitized[i].Description = strings.TrimSpace(sanitized[i].Description)
		sanitized[i].ImageURL = strings.TrimSpace(sanitized[i].ImageURL)
		if sanitized[i].ImageURL == "" {
			sanitized[i].ImageURL = DefaultHelpGroupCardImageURL
		}
	}
	return sanitized
}

func GetHelpGroupCards() []HelpGroupCard {
	if HelpGroupCards == nil {
		return sanitizeHelpGroupCards(defaultHelpGroupCards)
	}
	return sanitizeHelpGroupCards(HelpGroupCards)
}

func ValidateHelpGroupCards(cards []HelpGroupCard) ([]HelpGroupCard, error) {
	seenIDs := make(map[string]struct{}, len(cards))
	validated := make([]HelpGroupCard, 0, len(cards))
	for index, card := range cards {
		card.ID = strings.TrimSpace(card.ID)
		card.Label = strings.TrimSpace(card.Label)
		card.Title = strings.TrimSpace(card.Title)
		card.Description = strings.TrimSpace(card.Description)
		card.ImageURL = strings.TrimSpace(card.ImageURL)

		if card.ID == "" {
			return nil, fmt.Errorf("第 %d 张群聊卡片缺少 id", index+1)
		}
		if card.Title == "" {
			return nil, fmt.Errorf("第 %d 张群聊卡片缺少 title", index+1)
		}
		if _, exists := seenIDs[card.ID]; exists {
			return nil, fmt.Errorf("群聊卡片 id 重复: %s", card.ID)
		}
		seenIDs[card.ID] = struct{}{}
		validated = append(validated, card)
	}
	return validated, nil
}

func ValidateHelpGroupCardsJSON(jsonString string) ([]HelpGroupCard, error) {
	var cards []HelpGroupCard
	if err := common.UnmarshalJsonStr(jsonString, &cards); err != nil {
		return nil, err
	}
	return ValidateHelpGroupCards(cards)
}

func UpdateHelpGroupCardsByJSONString(jsonString string) error {
	cards, err := ValidateHelpGroupCardsJSON(jsonString)
	if err != nil {
		return err
	}
	HelpGroupCards = cards
	return nil
}

func HelpGroupCards2JSONString() string {
	jsonBytes, err := common.Marshal(GetHelpGroupCards())
	if err != nil {
		common.SysLog("error marshalling help group cards: " + err.Error())
		return "[]"
	}
	return string(jsonBytes)
}
