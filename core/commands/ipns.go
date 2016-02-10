package commands

import (
	"errors"
	"io"
	"strings"

	cmds "github.com/ipfs/go-ipfs/commands"
	namesys "github.com/ipfs/go-ipfs/namesys"
	offline "github.com/ipfs/go-ipfs/routing/offline"
	u "gx/ipfs/QmZNVWh8LLjAavuQ2JXuFmuYH3C11xo988vSgp7UQrTRj1/go-ipfs-util"
)

var IpnsCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Gets the value currently published at an IPNS name.",
		ShortDescription: `
IPNS is a PKI namespace, where names are the hashes of public keys, and
the private key enables publishing new (signed) values. In resolve, the
default value of <name> is your own identity public key.
`,
		LongDescription: `
IPNS is a PKI namespace, where names are the hashes of public keys, and
the private key enables publishing new (signed) values. In resolve, the
default value of <name> is your own identity public key.


Examples:

Resolve the value of your identity:

  > ipfs name resolve
  QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy

Resolve the value of another name:

  > ipfs name resolve QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n
  QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy

`,
	},

	Arguments: []cmds.Argument{
		cmds.StringArg("name", false, false, "The IPNS name to resolve. Defaults to your node's peerID.").EnableStdin(),
	},
	Options: []cmds.Option{
		cmds.BoolOption("recursive", "r", "Resolve until the result is not an IPNS name."),
		cmds.BoolOption("nocache", "n", "Do not used cached entries."),
	},
	Run: func(req cmds.Request, res cmds.Response) {

		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if !n.OnlineMode() {
			err := n.SetupOfflineRouting()
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
		}

		nocache, _, _ := req.Option("nocache").Bool()
		local, _, _ := req.Option("local").Bool()

		// default to nodes namesys resolver
		var resolver namesys.Resolver = n.Namesys

		if local && nocache {
			res.SetError(errors.New("cannot specify both local and nocache"), cmds.ErrNormal)
			return
		}

		if local {
			offroute := offline.NewOfflineRouter(n.Repo.Datastore(), n.PrivateKey)
			resolver = namesys.NewRoutingResolver(offroute, 0)
		}

		if nocache {
			resolver = namesys.NewNameSystem(n.Routing, n.Repo.Datastore(), 0)
		}

		var name string
		if len(req.Arguments()) == 0 {
			if n.Identity == "" {
				res.SetError(errors.New("Identity not loaded!"), cmds.ErrNormal)
				return
			}
			name = n.Identity.Pretty()

		} else {
			name = req.Arguments()[0]
		}

		recursive, _, _ := req.Option("recursive").Bool()
		depth := 1
		if recursive {
			depth = namesys.DefaultDepthLimit
		}

		if !strings.HasPrefix(name, "/ipns/") {
			name = "/ipns/" + name
		}

		output, err := resolver.ResolveN(req.Context(), name, depth)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		// TODO: better errors (in the case of not finding the name, we get "failed to find any peer in table")

		res.SetOutput(&ResolvedPath{output})
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			output, ok := res.Output().(*ResolvedPath)
			if !ok {
				return nil, u.ErrCast()
			}
			return strings.NewReader(output.Path.String() + "\n"), nil
		},
	},
	Type: ResolvedPath{},
}
