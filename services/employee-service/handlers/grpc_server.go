package handlers

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
	pb "github.com/RAF-SI-2025/EXBanka-4-Backend/shared/pb/employee"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EmployeeServer struct {
	pb.UnimplementedEmployeeServiceServer
	DB *sql.DB
}

const defaultPageSize = 20
const maxPageSize = 100

func paginate(page, pageSize int32) (limit, offset int32) {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	if page <= 0 {
		page = 1
	}
	return pageSize, (page - 1) * pageSize
}

func (s *EmployeeServer) GetAllEmployees(ctx context.Context, req *pb.GetAllEmployeesRequest) (*pb.GetAllEmployeesResponse, error) {
	limit, offset := paginate(req.Page, req.PageSize)

	var total int32
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM employees`).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, first_name, last_name, date_of_birth::text, gender, email,
		       phone_number, address, username, position, department, active, permissions, jmbg
		FROM employees
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []*pb.Employee
	for rows.Next() {
		var e pb.Employee
		var permissions pq.StringArray
		if err := rows.Scan(
			&e.Id, &e.FirstName, &e.LastName, &e.DateOfBirth, &e.Gender, &e.Email,
			&e.PhoneNumber, &e.Address, &e.Username, &e.Position,
			&e.Department, &e.Active, &permissions, &e.Jmbg,
		); err != nil {
			return nil, err
		}
		e.Permissions = permissions
		employees = append(employees, &e)
	}
	return &pb.GetAllEmployeesResponse{Employees: employees, TotalCount: total}, nil
}

func (s *EmployeeServer) SearchEmployees(ctx context.Context, req *pb.SearchEmployeesRequest) (*pb.SearchEmployeesResponse, error) {
	limit, offset := paginate(req.Page, req.PageSize)

	var total int32
	if err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM employees
		WHERE ($1 = '' OR email      ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR first_name ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR last_name  ILIKE '%' || $3 || '%')
		  AND ($4 = '' OR position   ILIKE '%' || $4 || '%')`,
		req.Email, req.FirstName, req.LastName, req.Position,
	).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, first_name, last_name, date_of_birth::text, gender, email,
		       phone_number, address, username, position, department, active, permissions, jmbg
		FROM employees
		WHERE ($1 = '' OR email      ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR first_name ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR last_name  ILIKE '%' || $3 || '%')
		  AND ($4 = '' OR position   ILIKE '%' || $4 || '%')
		LIMIT $5 OFFSET $6`,
		req.Email, req.FirstName, req.LastName, req.Position, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []*pb.Employee
	for rows.Next() {
		var e pb.Employee
		var permissions pq.StringArray
		if err := rows.Scan(
			&e.Id, &e.FirstName, &e.LastName, &e.DateOfBirth, &e.Gender, &e.Email,
			&e.PhoneNumber, &e.Address, &e.Username, &e.Position,
			&e.Department, &e.Active, &permissions, &e.Jmbg,
		); err != nil {
			return nil, err
		}
		e.Permissions = permissions
		employees = append(employees, &e)
	}
	return &pb.SearchEmployeesResponse{Employees: employees, TotalCount: total}, nil
}

func (s *EmployeeServer) GetEmployeeCredentials(ctx context.Context, req *pb.GetEmployeeCredentialsRequest) (*pb.GetEmployeeCredentialsResponse, error) {
	var id int64
	var passwordHash string
	var active bool
	var permissions pq.StringArray
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, password, active, permissions FROM employees WHERE email = $1`,
		req.Email,
	).Scan(&id, &passwordHash, &active, &permissions)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, err
	}
	return &pb.GetEmployeeCredentialsResponse{Id: id, PasswordHash: passwordHash, Active: active, Permissions: permissions}, nil
}

func (s *EmployeeServer) GetEmployeeById(ctx context.Context, req *pb.GetEmployeeByIdRequest) (*pb.GetEmployeeByIdResponse, error) {
	var e pb.Employee
	var permissions pq.StringArray
	err := s.DB.QueryRowContext(ctx, `
		SELECT id, first_name, last_name, date_of_birth::text, gender, email,
		       phone_number, address, username, position, department, active, permissions, jmbg
		FROM employees WHERE id = $1`, req.Id,
	).Scan(
		&e.Id, &e.FirstName, &e.LastName, &e.DateOfBirth, &e.Gender, &e.Email,
		&e.PhoneNumber, &e.Address, &e.Username, &e.Position,
		&e.Department, &e.Active, &permissions, &e.Jmbg,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		return nil, err
	}
	e.Permissions = permissions
	return &pb.GetEmployeeByIdResponse{Employee: &e}, nil
}

func (s *EmployeeServer) UpdateEmployee(ctx context.Context, req *pb.UpdateEmployeeRequest) (*pb.UpdateEmployeeResponse, error) {
	if req.Active {
		var pwd string
		err := s.DB.QueryRowContext(ctx, `SELECT password FROM employees WHERE id = $1`, req.Id).Scan(&pwd)
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "employee not found")
		}
		if err != nil {
			return nil, err
		}
		if pwd == "" {
			return nil, status.Error(codes.FailedPrecondition, "cannot activate employee: no password set")
		}
	}

	var e pb.Employee
	var permissions pq.StringArray
	err := s.DB.QueryRowContext(ctx, `
		UPDATE employees
		SET first_name=$2, last_name=$3, date_of_birth=$4::date, gender=$5, email=$6,
		    phone_number=$7, address=$8, username=$9, position=$10,
		    department=$11, active=$12, permissions=$13, jmbg=$14
		WHERE id=$1
		RETURNING id, first_name, last_name, date_of_birth::text, gender, email,
		          phone_number, address, username, position, department, active, permissions, jmbg`,
		req.Id, req.FirstName, req.LastName, req.DateOfBirth, req.Gender, req.Email,
		req.PhoneNumber, req.Address, req.Username, req.Position,
		req.Department, req.Active, pq.StringArray(req.Permissions), req.Jmbg,
	).Scan(
		&e.Id, &e.FirstName, &e.LastName, &e.DateOfBirth, &e.Gender, &e.Email,
		&e.PhoneNumber, &e.Address, &e.Username, &e.Position,
		&e.Department, &e.Active, &permissions, &e.Jmbg,
	)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			switch pqErr.Constraint {
			case "employees_username_key":
				return nil, status.Error(codes.AlreadyExists, "username already exists")
			case "employees_jmbg_key":
				return nil, status.Error(codes.AlreadyExists, "jmbg already exists")
			default:
				return nil, status.Error(codes.AlreadyExists, "email already exists")
			}
		}
		return nil, err
	}
	e.Permissions = permissions
	return &pb.UpdateEmployeeResponse{Employee: &e}, nil
}

func (s *EmployeeServer) ActivateEmployee(ctx context.Context, req *pb.ActivateEmployeeRequest) (*pb.ActivateEmployeeResponse, error) {
	var active bool
	var pwd string
	err := s.DB.QueryRowContext(ctx, `SELECT active, password FROM employees WHERE id = $1`, req.EmployeeId).Scan(&active, &pwd)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	if err != nil {
		return nil, err
	}
	if active || pwd != "" {
		return nil, status.Error(codes.FailedPrecondition, "employee already activated")
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE employees SET password = $2, active = true WHERE id = $1`, req.EmployeeId, req.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &pb.ActivateEmployeeResponse{}, nil
}

func (s *EmployeeServer) GetEmployeeByEmail(ctx context.Context, req *pb.GetEmployeeByEmailRequest) (*pb.GetEmployeeByEmailResponse, error) {
	var id int64
	var firstName, email string
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, first_name, email FROM employees WHERE email = $1`,
		req.Email,
	).Scan(&id, &firstName, &email)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "user with this email doesn't exist")
	}
	if err != nil {
		return nil, err
	}
	return &pb.GetEmployeeByEmailResponse{Id: id, FirstName: firstName, Email: email}, nil
}

func (s *EmployeeServer) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*pb.UpdatePasswordResponse, error) {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE employees SET password = $2 WHERE id = $1`,
		req.EmployeeId, req.PasswordHash,
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, status.Error(codes.NotFound, "employee not found")
	}
	return &pb.UpdatePasswordResponse{}, nil
}

func (s *EmployeeServer) CreateEmployee(ctx context.Context, req *pb.CreateEmployeeRequest) (*pb.CreateEmployeeResponse, error) {
	var id int64
	err := s.DB.QueryRowContext(ctx, `
		INSERT INTO employees
			(first_name, last_name, date_of_birth, gender, email, phone_number, address, username,
			 password, position, department, active, permissions, jmbg)
		VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, '', $9, $10, false, '{}', $11)
		RETURNING id`,
		req.FirstName, req.LastName, req.DateOfBirth, req.Gender, req.Email,
		req.PhoneNumber, req.Address, req.Username, req.Position, req.Department, req.Jmbg,
	).Scan(&id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			switch pqErr.Constraint {
			case "employees_username_key":
				return nil, status.Error(codes.AlreadyExists, "username already exists")
			case "employees_jmbg_key":
				return nil, status.Error(codes.AlreadyExists, "jmbg already exists")
			default:
				return nil, status.Error(codes.AlreadyExists, "email already exists")
			}
		}
		return nil, err
	}
	return &pb.CreateEmployeeResponse{
		Employee: &pb.Employee{
			Id:          id,
			FirstName:   req.FirstName,
			LastName:    req.LastName,
			DateOfBirth: req.DateOfBirth,
			Gender:      req.Gender,
			Email:       req.Email,
			PhoneNumber: req.PhoneNumber,
			Address:     req.Address,
			Username:    req.Username,
			Position:    req.Position,
			Department:  req.Department,
			Active:      false,
			Permissions: []string{},
			Jmbg:        req.Jmbg,
		},
	}, nil
}
