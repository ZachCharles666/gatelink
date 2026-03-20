package db

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// 业务错误类型
var (
	ErrNotFound            = errors.New("record not found")
	ErrDuplicate           = errors.New("duplicate record")
	ErrConstraint          = errors.New("constraint violation")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// IsNotFound 判断是否为记录不存在错误
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrNotFound)
}

// IsDuplicate 判断是否为唯一约束冲突
func IsDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // unique_violation
	}
	return false
}

// IsConstraintViolation 判断是否为约束冲突
func IsConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return strings.HasPrefix(pgErr.Code, "23")
	}
	return false
}

// WrapPgError 将 pg 错误转换为业务错误
func WrapPgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if IsDuplicate(err) {
		return ErrDuplicate
	}
	if IsConstraintViolation(err) {
		return ErrConstraint
	}
	return err
}
