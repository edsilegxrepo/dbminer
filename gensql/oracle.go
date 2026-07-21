package gensql

type OracleDriver struct{}

func (d *OracleDriver) Name() string { return "oracle" }

func (d *OracleDriver) GenerateSQL(opts GenerateOptions) (string, error) {
	vars := TemplateVars{
		IncludeAdmin: !opts.NoAdmin,
		StoredProc:   false, // Oracle uses direct SQL
		DBName:       opts.DBName,
		SPName:       opts.SPName,
	}

	sql, err := ExecuteTemplate("oracle", vars)
	if err != nil {
		return "", err
	}

	return WrapWithUser(sql, opts.ExecUser, "oracle"), nil
}
