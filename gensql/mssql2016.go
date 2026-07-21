package gensql

type MSSQL2016Driver struct{}

func (d *MSSQL2016Driver) Name() string { return "mssql2016" }

func (d *MSSQL2016Driver) GenerateSQL(opts GenerateOptions) (string, error) {
	vars := TemplateVars{
		IncludeAdmin: !opts.NoAdmin,
		StoredProc:   !opts.Direct,
		DBName:       opts.DBName,
		SPName:       opts.SPName,
	}

	sql, err := ExecuteTemplate("mssql2016", vars)
	if err != nil {
		return "", err
	}

	return WrapWithUser(sql, opts.ExecUser, "mssql2016"), nil
}
