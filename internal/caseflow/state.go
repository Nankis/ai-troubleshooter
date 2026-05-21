package caseflow

var allowedTransitions = map[Status]map[Status]bool{
	StatusNew: {
		StatusNeedMoreInfo:       true,
		StatusReadyToInvestigate: true,
		StatusFailed:             true,
		StatusCancelled:          true,
	},
	StatusNeedMoreInfo: {
		StatusWaitingUserReply:   true,
		StatusReadyToInvestigate: true,
		StatusFailed:             true,
		StatusCancelled:          true,
	},
	StatusWaitingUserReply: {
		StatusReadyToInvestigate: true,
		StatusFailed:             true,
		StatusCancelled:          true,
	},
	StatusReadyToInvestigate: {
		StatusInvestigating: true,
		StatusFailed:        true,
		StatusCancelled:     true,
	},
	StatusInvestigating: {
		StatusWaitingToolResult:     true,
		StatusNeedHumanConfirmation: true,
		StatusDone:                  true,
		StatusFailed:                true,
	},
	StatusWaitingToolResult: {
		StatusInvestigating:         true,
		StatusNeedHumanConfirmation: true,
		StatusDone:                  true,
		StatusFailed:                true,
	},
	StatusNeedHumanConfirmation: {
		StatusDone:   true,
		StatusFailed: true,
	},
}

func CanTransition(from, to Status) bool {
	if from == to {
		return true
	}
	next, ok := allowedTransitions[from]
	return ok && next[to]
}

func MissingRequiredFields(domain string, entities map[string]string) []string {
	required := []string{}
	switch domain {
	case DomainKline:
		required = []string{"symbol", "interval", "abnormal_time", "issue_type"}
	case DomainAsset:
		if entities["user_id"] == "" && entities["account_id"] == "" {
			required = append(required, "user_id 或 account_id")
		}
		required = append(required, "asset_symbol", "abnormal_time", "issue_type")
	default:
		return []string{"issue_domain"}
	}

	missing := make([]string, 0, len(required))
	for _, field := range required {
		if field == "user_id 或 account_id" {
			missing = append(missing, field)
			continue
		}
		if entities[field] == "" {
			missing = append(missing, field)
		}
	}
	return missing
}

func EntityMap(entities []Entity) map[string]string {
	out := make(map[string]string, len(entities))
	for _, entity := range entities {
		if _, exists := out[entity.Type]; !exists && entity.Value != "" {
			out[entity.Type] = entity.Value
		}
	}
	return out
}
