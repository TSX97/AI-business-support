package corehttp

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const businessProfileType = "company_profile"

var errBusinessNotOwned = errors.New("caller does not own business")

type businessCursor struct {
	UpdatedAt  time.Time `json:"updated_at"`
	BusinessID int64     `json:"business_id"`
}

func insertBusiness(
	ctx context.Context,
	db chatDB,
	userID int64,
	name string,
	description *string,
	businessData json.RawMessage,
) (businessRecord, error) {
	var business businessRecord
	var nullableDescription sql.NullString
	err := db.QueryRow(
		ctx,
		`WITH new_company AS (
		   INSERT INTO core.companies (name, description)
		   VALUES ($2, $3)
		   RETURNING id, name, description, created_at
		 ),
		 new_owner AS (
		   INSERT INTO core.company_members (user_id, company_id, role)
		   SELECT $1, id, 'owner' FROM new_company
		   RETURNING company_id
		 ),
		 new_profile AS (
		   INSERT INTO core.business (company_id, data_type, data)
		   SELECT company_id, $4, $5::jsonb FROM new_owner
		   RETURNING id, company_id, data, created_at, updated_at
		 )
		 SELECT
		   profile.id,
		   $1::bigint,
		   company.name,
		   company.description,
		   profile.data,
		   profile.created_at,
		   profile.updated_at
		 FROM new_profile AS profile
		 JOIN new_company AS company ON company.id = profile.company_id`,
		userID,
		name,
		description,
		businessProfileType,
		[]byte(businessData),
	).Scan(
		&business.BusinessID,
		&business.UserID,
		&business.BusinessName,
		&nullableDescription,
		&business.BusinessData,
		&business.CreatedAt,
		&business.UpdatedAt,
	)
	if nullableDescription.Valid {
		business.Description = &nullableDescription.String
	}
	return business, err
}

func selectBusinesses(
	ctx context.Context,
	db chatDB,
	userID int64,
	cursor *businessCursor,
	limit int,
) ([]businessRecord, *string, error) {
	var encodedBusinesses []byte
	var cursorTime any
	var cursorID any
	if cursor != nil {
		cursorTime = cursor.UpdatedAt
		cursorID = cursor.BusinessID
	}

	err := db.QueryRow(
		ctx,
		`SELECT COALESCE(json_agg(row_to_json(business_row)), '[]'::json)
		 FROM (
		   SELECT
		     profile.id AS business_id,
		     owner.user_id,
		     company.name AS business_name,
		     company.description,
		     profile.data AS business_data,
		     profile.created_at,
		     profile.updated_at
		   FROM core.business AS profile
		   JOIN core.companies AS company ON company.id = profile.company_id
		   JOIN core.company_members AS membership ON membership.company_id = company.id
		   JOIN core.company_members AS owner ON owner.company_id = company.id AND owner.role = 'owner'
		   WHERE profile.data_type = $2
		     AND membership.user_id = $1
		     AND ($3::timestamp IS NULL OR (profile.updated_at, profile.id) < ($3::timestamp, $4::integer))
		   ORDER BY profile.updated_at DESC, profile.id DESC
		   LIMIT $5
		 ) AS business_row`,
		userID,
		businessProfileType,
		cursorTime,
		cursorID,
		limit+1,
	).Scan(&encodedBusinesses)
	if err != nil {
		return nil, nil, err
	}

	businesses := make([]businessRecord, 0)
	if err := json.Unmarshal(encodedBusinesses, &businesses); err != nil {
		return nil, nil, fmt.Errorf("decode business list: %w", err)
	}

	var nextCursor *string
	if len(businesses) > limit {
		businesses = businesses[:limit]
		encoded, err := encodeBusinessCursor(businessCursor{
			UpdatedAt:  businesses[len(businesses)-1].UpdatedAt,
			BusinessID: businesses[len(businesses)-1].BusinessID,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("encode business cursor: %w", err)
		}
		nextCursor = &encoded
	}
	return businesses, nextCursor, nil
}

func updateBusinessRecord(
	ctx context.Context,
	db chatDB,
	request updateBusinessRequest,
	description *string,
) (businessRecord, error) {
	var business businessRecord
	var nullableDescription sql.NullString
	err := db.QueryRow(
		ctx,
		`WITH target AS (
		   SELECT company.id
		   FROM core.business AS profile
		   JOIN core.companies AS company ON company.id = profile.company_id
		   JOIN core.company_members AS owner
		     ON owner.company_id = company.id AND owner.user_id = $2 AND owner.role = 'owner'
		   WHERE profile.id = $1 AND profile.data_type = $3
		 ),
		 updated_company AS (
		   UPDATE core.companies AS company
		   SET name = $4, description = $5
		   FROM target
		   WHERE company.id = target.id
		   RETURNING company.id, company.name, company.description
		 ),
		 updated_profile AS (
		   UPDATE core.business AS profile
		   SET data = $6::jsonb, updated_at = NOW()
		   FROM updated_company
		   WHERE profile.company_id = updated_company.id AND profile.data_type = $3
		   RETURNING profile.id, profile.company_id, profile.data, profile.created_at, profile.updated_at
		 )
		 SELECT
		   profile.id,
		   $2::bigint,
		   company.name,
		   company.description,
		   profile.data,
		   profile.created_at,
		   profile.updated_at
		 FROM updated_profile AS profile
		 JOIN updated_company AS company ON company.id = profile.company_id`,
		request.BusinessID,
		request.UserID,
		businessProfileType,
		request.BusinessName,
		description,
		[]byte(request.BusinessData),
	).Scan(
		&business.BusinessID,
		&business.UserID,
		&business.BusinessName,
		&nullableDescription,
		&business.BusinessData,
		&business.CreatedAt,
		&business.UpdatedAt,
	)
	if err == nil {
		if nullableDescription.Valid {
			business.Description = &nullableDescription.String
		}
		return business, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return businessRecord{}, err
	}

	_, owner, lookupErr := businessCompanyForUser(ctx, db, request.BusinessID, request.UserID)
	if lookupErr != nil {
		return businessRecord{}, lookupErr
	}
	if !owner {
		return businessRecord{}, errBusinessNotOwned
	}
	return businessRecord{}, pgx.ErrNoRows
}

func deleteBusinessRecord(ctx context.Context, db chatDB, businessID, userID int64) (int64, error) {
	var deletedBusinessID int64
	err := db.QueryRow(
		ctx,
		`WITH target AS MATERIALIZED (
		   SELECT profile.id AS business_id, profile.company_id
		   FROM core.business AS profile
		   JOIN core.company_members AS owner
		     ON owner.company_id = profile.company_id
		    AND owner.user_id = $2
		    AND owner.role = 'owner'
		   WHERE profile.id = $1 AND profile.data_type = $3
		 ),
		 deleted_profile AS (
		   DELETE FROM core.business AS profile
		   USING target
		   WHERE profile.id = target.business_id
		   RETURNING profile.company_id
		 ),
		 deleted_members AS (
		   DELETE FROM core.company_members AS member
		   USING target
		   WHERE member.company_id = target.company_id
		       AND NOT EXISTS (
		         SELECT 1
		         FROM core.business AS remaining
		         WHERE remaining.company_id = target.company_id
		           AND remaining.id <> target.business_id
		       )
		     RETURNING member.id
		   ),
		   detached_chats AS (
		     UPDATE core.chats AS chat
		     SET company_id = NULL
		     FROM target
		     WHERE chat.company_id = target.company_id
		       AND NOT EXISTS (
		         SELECT 1
		         FROM core.business AS remaining
		         WHERE remaining.company_id = target.company_id
		           AND remaining.id <> target.business_id
		       )
		     RETURNING chat.id
		   ),
		   deleted_company AS (
		     DELETE FROM core.companies AS company
		     USING target
		     WHERE company.id = target.company_id
		       AND NOT EXISTS (
		         SELECT 1
		         FROM core.business AS remaining
		         WHERE remaining.company_id = target.company_id
		           AND remaining.id <> target.business_id
		       )
		       AND (SELECT count(*) FROM deleted_profile) >= 0
		     AND (SELECT count(*) FROM deleted_members) >= 0
		     AND (SELECT count(*) FROM detached_chats) >= 0
		   RETURNING company.id
		 )
		 SELECT target.business_id
		 FROM target
		 CROSS JOIN (SELECT count(*) FROM deleted_company) AS cleanup`,
		businessID,
		userID,
		businessProfileType,
	).Scan(&deletedBusinessID)
	if err == nil {
		return deletedBusinessID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	_, owner, lookupErr := businessCompanyForUser(ctx, db, businessID, userID)
	if lookupErr != nil {
		return 0, lookupErr
	}
	if !owner {
		return 0, errBusinessNotOwned
	}
	return 0, pgx.ErrNoRows
}

func addBusinessMemberRecord(ctx context.Context, db chatDB, request businessMemberRequest) error {
	err := db.QueryRow(
		ctx,
		`INSERT INTO core.company_members (user_id, company_id, role)
		 SELECT $3, profile.company_id, $4
		 FROM core.business AS profile
		 JOIN core.company_members AS owner
		   ON owner.company_id = profile.company_id
		  AND owner.user_id = $2
		  AND owner.role = 'owner'
		 WHERE profile.id = $1 AND profile.data_type = $5
		 RETURNING id`,
		request.BusinessID,
		request.UserID,
		request.MemberUserID,
		request.Role,
		businessProfileType,
	).Scan(new(int64))
	if errors.Is(err, pgx.ErrNoRows) {
		_, owner, lookupErr := businessCompanyForUser(ctx, db, request.BusinessID, request.UserID)
		if lookupErr != nil {
			return lookupErr
		}
		if !owner {
			return errBusinessNotOwned
		}
	}
	return err
}

func removeBusinessMemberRecord(ctx context.Context, db chatDB, request businessMemberDeleteRequest) error {
	var removedMemberID int64
	err := db.QueryRow(
		ctx,
		`DELETE FROM core.company_members AS member
		 USING core.business AS profile, core.company_members AS owner
		 WHERE profile.id = $1
		   AND profile.data_type = $4
		   AND member.company_id = profile.company_id
		   AND member.user_id = $3
		   AND member.role <> 'owner'
		   AND owner.company_id = profile.company_id
		   AND owner.user_id = $2
		   AND owner.role = 'owner'
		 RETURNING user_id`,
		request.BusinessID,
		request.UserID,
		request.MemberUserID,
		businessProfileType,
	).Scan(&removedMemberID)
	if errors.Is(err, pgx.ErrNoRows) {
		_, owner, lookupErr := businessCompanyForUser(ctx, db, request.BusinessID, request.UserID)
		if lookupErr != nil {
			return lookupErr
		}
		if !owner {
			return errBusinessNotOwned
		}
	}
	return err
}

func businessCompanyForUser(ctx context.Context, db chatDB, businessID, userID int64) (int64, bool, error) {
	var companyID int64
	var isOwner bool
	err := db.QueryRow(
		ctx,
		`SELECT company.id,
		        EXISTS (
		          SELECT 1 FROM core.company_members AS owner
		          WHERE owner.company_id = company.id
		            AND owner.user_id = $2
		            AND owner.role = 'owner'
		        )
		 FROM core.business AS profile
		 JOIN core.companies AS company ON company.id = profile.company_id
		 WHERE profile.id = $1 AND profile.data_type = $3`,
		businessID,
		userID,
		businessProfileType,
	).Scan(&companyID, &isOwner)
	return companyID, isOwner, err
}

func encodeBusinessCursor(cursor businessCursor) (string, error) {
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func parseBusinessCursor(value string) (*businessCursor, error) {
	if value == "" {
		return nil, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	var cursor businessCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, err
	}
	if cursor.BusinessID <= 0 || cursor.UpdatedAt.IsZero() {
		return nil, errors.New("cursor is missing updated_at or business_id")
	}
	return &cursor, nil
}
