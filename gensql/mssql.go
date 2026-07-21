package gensql

type MSSQLDriver struct{}

func (d *MSSQLDriver) Name() string { return "mssql" }

func (d *MSSQLDriver) GenerateSQL(opts GenerateOptions) (string, error) {
	vars := TemplateVars{
		IncludeAdmin: !opts.NoAdmin,
		StoredProc:   !opts.Direct,
		DBName:       opts.DBName,
		SPName:       opts.SPName,
	}

	sql, err := ExecuteTemplate("mssql", vars)
	if err != nil {
		return "", err
	}

	return WrapWithUser(sql, opts.ExecUser, "mssql"), nil
}
