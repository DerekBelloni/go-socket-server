package store

import (
	"reflect"
	"testing"
)

func TestExtractPubKey(t *testing.T) {
	// Slice of p tags
	t.Run("Valid input", func(t *testing.T) {
		validTags := []interface{}{
			[]interface{}{"p", "2250f69694c2a43929e77e5de0f6a61ae5e37a1ee6d6a3baef1706ed9901248b"},
			[]interface{}{"p", "6c535d95a8659b234d5a0805034f5f0a67e3c0ceffcc459f61f680fe944424bf"},
			[]interface{}{"p", "84dee6e676e5bb67b4ad4e042cf70cbd8681155db535942fcc6a0533858a7240"},
			[]interface{}{"p", "82341f882b6eabcd2ba7f1ef90aad961cf074af15b9ef44a09f9d2a8fbfbe6a2"},
			[]interface{}{"p", "9d065f84c0cba7b0ef86f5d2d155e6ce01178a8a33e194f9999b7497b1b2201b"},
		}
		result, ok := ExtractPubKeys(validTags)
		expected := []string{
			"2250f69694c2a43929e77e5de0f6a61ae5e37a1ee6d6a3baef1706ed9901248b",
			"6c535d95a8659b234d5a0805034f5f0a67e3c0ceffcc459f61f680fe944424bf",
			"84dee6e676e5bb67b4ad4e042cf70cbd8681155db535942fcc6a0533858a7240",
			"82341f882b6eabcd2ba7f1ef90aad961cf074af15b9ef44a09f9d2a8fbfbe6a2",
			"9d065f84c0cba7b0ef86f5d2d155e6ce01178a8a33e194f9999b7497b1b2201b",
		}

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("ExtractPubKeys = %v; want: %v\n", result, expected)
		}

		if !ok {
			t.Errorf("ExtractPubKeys(%v) returned false, expected true", validTags)
		}
	})
	t.Run("Empty input", func(t *testing.T) {
		emptyTags := []interface{}{}
		result, ok := ExtractPubKeys(emptyTags)

		if result != nil {
			t.Errorf("ExtractPubKeys(%v) returned %v, expected nil", emptyTags, result)
		}

		if ok {
			t.Errorf("ExtractPubKeys(%v) returned true, expected false", emptyTags)
		}
	})

	t.Run("Invalid tag format", func(t *testing.T) {
		invalidTags := []interface{}{
			[]interface{}{"e", "2250f69694c2a43929e77e5de0f6a61ae5e37a1ee6d6a3baef1706ed9901248b"},
			[]interface{}{"e", "6c535d95a8659b234d5a0805034f5f0a67e3c0ceffcc459f61f680fe944424bf"},
			[]interface{}{"e", "84dee6e676e5bb67b4ad4e042cf70cbd8681155db535942fcc6a0533858a7240"},
			[]interface{}{"e", "82341f882b6eabcd2ba7f1ef90aad961cf074af15b9ef44a09f9d2a8fbfbe6a2"},
			[]interface{}{"e", "9d065f84c0cba7b0ef86f5d2d155e6ce01178a8a33e194f9999b7497b1b2201b"},
		}
		result, ok := ExtractPubKeys(invalidTags)

		if result != nil {
			t.Errorf("ExtractPubKeys(%v) returned %v, expected nil", invalidTags, result)
		}

		if ok {
			t.Errorf("ExtractPubKeys(%v) returned true, expected false", invalidTags)
		}
	})

	t.Run("Mixed valid and invalid tags", func(t *testing.T) {
		mixedTags := []interface{}{
			[]interface{}{"p", "2250f69694c2a43929e77e5de0f6a61ae5e37a1ee6d6a3baef1706ed9901248b"},
			[]interface{}{"p", ""},
			"not a slice",
			[]interface{}{"p", 12341},
			[]interface{}{"p", "9d065f84c0cba7b0ef86f5d2d155e6ce01178a8a33e194f9999b7497b1b2201b"},
		}

		result, ok := ExtractPubKeys(mixedTags)

		expected := []string{
			"2250f69694c2a43929e77e5de0f6a61ae5e37a1ee6d6a3baef1706ed9901248b",
			"9d065f84c0cba7b0ef86f5d2d155e6ce01178a8a33e194f9999b7497b1b2201b",
		}

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("ExtractPubKeys = %v; want: %v\n", result, expected)
		}

		if !ok {
			t.Errorf("ExtractPubKeys(%v) returned false, expected true", mixedTags)
		}
	})
}
