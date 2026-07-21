package gensql

type SQLiteDriver struct{}

func (d *SQLiteDriver) Name() string { return "sqlite" }

func (d *SQLiteDriver) GenerateSQL(opts GenerateOptions) (string, error) {
	vars := TemplateVars{
		IncludeAdmin: !opts.NoAdmin,
		StoredProc:   false, // SQLite doesn't support stored procedures
		DBName:       opts.DBName,
		SPName:       opts.SPName,
	}

	sql, err := ExecuteTemplate("sqlite", vars)
	if err != nil {
		return "", err
	}

	return WrapWithUser(sql, opts.ExecUser, "sqlite"), nil
}
