package gensql

type PostgresDriver struct{}

func (d *PostgresDriver) Name() string { return "postgres" }

func (d *PostgresDriver) GenerateSQL(opts GenerateOptions) (string, error) {
	vars := TemplateVars{
		IncludeAdmin: !opts.NoAdmin,
		StoredProc:   false, // PostgreSQL doesn't use stored proc wrapper
		DBName:       opts.DBName,
		SPName:       opts.SPName,
	}

	sql, err := ExecuteTemplate("postgres", vars)
	if err != nil {
		return "", err
	}

	return WrapWithUser(sql, opts.ExecUser, "postgres"), nil
}
