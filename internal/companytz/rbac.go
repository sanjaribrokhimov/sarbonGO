package companytz

// CanInvite returns true if role can invite the target role.
// Owner: any; CEO: any except Owner; TopManager: Manager; TopDispatcher: Dispatcher.
func CanInvite(actorRole, targetRoleName string) bool {
	switch actorRole {
	case "Owner":
		return true
	case "CEO":
		return targetRoleName != "Owner"
	case "TopManager":
		return targetRoleName == "Manager"
	case "TopDispatcher":
		return targetRoleName == "Dispatcher"
	default:
		return false
	}
}

// CanChangeRole returns true if actor can assign targetRole to someone.
func CanChangeRole(actorRole, targetRoleName string) bool {
	return CanInvite(actorRole, targetRoleName)
}

// CanRemove returns true if actor can remove a user with targetRole from company.
func CanRemove(actorRole, targetRoleName string) bool {
	switch actorRole {
	case "Owner":
		return true
	case "CEO":
		return targetRoleName != "Owner"
	case "TopManager":
		return targetRoleName == "Manager"
	case "TopDispatcher":
		return targetRoleName == "Dispatcher"
	default:
		return false
	}
}
