# twi/fix

Every exported name in `twi/fix`, one line each, grouped by the file that declares it. A test reads them from the Go source and fails when this page drifts from it.

## fix.go

type Plan struct\
Plan.Terminal string\
Plan.Arch string\
Plan.Backups \[\]string\
Plan.Package string\
Plan.SHA256 string\
func (Plan) String() string\
type Refused string\
func (Refused) Error() string\
func ConPTY(ctx context.Context, ask func(Plan) bool) (string, error)\
func Undo(dir string) (string, error)\
func Ask(r io.Reader, w io.Writer) func(Plan) bool
