package middleware

import "smartbed/internal/domain"

// RoleOperatorAll returns roles allowed to ingest vitals data.
func RoleOperatorAll() []domain.Role {
	return []domain.Role{domain.RoleOperator, domain.RoleAdmin}
}

// RoleClinicalAll returns roles with access to clinical read data.
func RoleClinicalAll() []domain.Role {
	return []domain.Role{domain.RoleAdmin, domain.RoleDoctor, domain.RoleCaregiver}
}

// RoleOnlyDoctor returns the doctor-only role set.
func RoleOnlyDoctor() []domain.Role {
	return []domain.Role{domain.RoleDoctor}
}

// RoleAdminDoctor returns admin + doctor roles.
func RoleAdminDoctor() []domain.Role {
	return []domain.Role{domain.RoleAdmin, domain.RoleDoctor}
}
