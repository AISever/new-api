package setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetHelpGroupCards_DefaultsWhenEmpty(t *testing.T) {
	original := HelpGroupCards
	t.Cleanup(func() {
		HelpGroupCards = original
	})

	HelpGroupCards = nil

	cards := GetHelpGroupCards()
	require.Len(t, cards, 1)
	require.Equal(t, "/WeChat-qun.jpg", cards[0].ImageURL)
	require.True(t, cards[0].Enabled)
}

func TestUpdateHelpGroupCardsByJSONString_AcceptsValidCards(t *testing.T) {
	original := HelpGroupCards
	t.Cleanup(func() {
		HelpGroupCards = original
	})

	err := UpdateHelpGroupCardsByJSONString(`[
	  {
	    "id": "group-a",
	    "enabled": true,
	    "label": "群聊：A",
	    "title": "A 群",
	    "description": "描述 A",
	    "image_url": "/site-assets/help-group-card/a.png",
	    "sort_order": 2
	  },
	  {
	    "id": "group-b",
	    "enabled": false,
	    "label": "群聊：B",
	    "title": "B 群",
	    "description": "描述 B",
	    "image_url": "/site-assets/help-group-card/b.png",
	    "sort_order": 1
	  }
	]`)

	require.NoError(t, err)
	require.Len(t, GetHelpGroupCards(), 2)
	require.Equal(t, "group-a", HelpGroupCards[0].ID)
	require.Equal(t, 2, HelpGroupCards[0].SortOrder)
}

func TestUpdateHelpGroupCardsByJSONString_RejectsMissingRequiredFields(t *testing.T) {
	_, err := ValidateHelpGroupCardsJSON(`[
	  {
	    "id": "group-a",
	    "enabled": true,
	    "label": "群聊：A",
	    "description": "描述 A",
	    "image_url": "/site-assets/help-group-card/a.png",
	    "sort_order": 2
	  }
	]`)

	require.Error(t, err)
	require.ErrorContains(t, err, "title")
}

func TestUpdateHelpGroupCardsByJSONString_RejectsDuplicateIDs(t *testing.T) {
	_, err := ValidateHelpGroupCardsJSON(`[
	  {
	    "id": "group-a",
	    "enabled": true,
	    "label": "群聊：A",
	    "title": "A 群",
	    "description": "描述 A",
	    "image_url": "/site-assets/help-group-card/a.png",
	    "sort_order": 2
	  },
	  {
	    "id": "group-a",
	    "enabled": true,
	    "label": "群聊：B",
	    "title": "B 群",
	    "description": "描述 B",
	    "image_url": "/site-assets/help-group-card/b.png",
	    "sort_order": 3
	  }
	]`)

	require.Error(t, err)
	require.ErrorContains(t, err, "id")
}
