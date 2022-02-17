package trcinitbase

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	il "tierceron/trcinit/initlib"
	"tierceron/utils"
	eUtils "tierceron/utils"
	"tierceron/vaulthelper/kv"
	sys "tierceron/vaulthelper/system"
)

// This assumes that the vault is completely new, and should only be run for the purpose
// of automating setup and initial seeding

func CommonMain(envPtr *string, addrPtrIn *string) {
	devPtr := flag.Bool("dev", false, "Vault server running in dev mode (does not need to be unsealed)")
	newPtr := flag.Bool("new", false, "New vault being initialized. Creates engines and requests first-time initialization")
	addrPtr := flag.String("addr", "", "API endpoint for the vault")
	if addrPtrIn != nil && *addrPtrIn != "" {
		addrPtr = addrPtrIn
	}
	seedPtr := flag.String("seeds", "trc_seeds", "Directory that contains vault seeds")
	tokenPtr := flag.String("token", "", "Vault access token, only use if in dev mode or reseeding")
	shardPtr := flag.String("shard", "", "Key shard used to unseal a vault that has been initialized but restarted")

	namespaceVariable := flag.String("namespace", "", "name of the namespace")

	logFilePtr := flag.String("log", "./trcinit.log", "Output path for log files")
	servicePtr := flag.String("service", "", "Seeding vault with a single service")
	prodPtr := flag.Bool("prod", false, "Prod only seeds vault with staging environment")
	uploadCertPtr := flag.Bool("certs", false, "Upload certs if provided")
	rotateTokens := flag.Bool("rotateTokens", false, "rotate tokens")
	tokenExpiration := flag.Bool("tokenExpiration", false, "Look up Token expiration dates")
	pingPtr := flag.Bool("ping", false, "Ping vault.")
	updateRole := flag.Bool("updateRole", false, "Update security role")
	updatePolicy := flag.Bool("updatePolicy", false, "Update security policy")
	initNamespace := flag.Bool("initns", false, "Init namespace (tokens, policy, and role)")
	insecurePtr := flag.Bool("insecure", false, "By default, every ssl connection is secure.  Allows to continue with server connections considered insecure.")
	keyShardPtr := flag.String("totalKeys", "5", "Total number of key shards to make")
	unsealShardPtr := flag.String("unsealKeys", "3", "Number of key shards needed to unseal")
	indexPtr := flag.String("index", "", "Specifies which projects are indexed")
	restrictedPtr := flag.String("restricted", "", "Specfies which projects have restricted access.")

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		s := args[i]
		if s[0] != '-' {
			fmt.Println("Wrong flag syntax: ", s)
			os.Exit(1)
		}
	}
	flag.Parse()

	// Prints usage if no flags are specified
	if flag.NFlag() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if *namespaceVariable == "" && *newPtr {
		fmt.Println("Namespace (-namespace) required to initialize a new vault.")
		os.Exit(1)
	}

	if len(*indexPtr) > 0 && len(*restrictedPtr) > 0 {
		fmt.Println("-index and -restricted flag cannot be used together.")
		os.Exit(1)
	}

	var indexSlice = make([]string, 0) //Checks for indexed projects
	if len(*indexPtr) > 0 {
		indexSlice = append(indexSlice, strings.Split(*indexPtr, ",")...)
	}

	var restrictedSlice = make([]string, 0) //Checks for restricted projects
	if len(*restrictedPtr) > 0 {
		restrictedSlice = append(restrictedSlice, strings.Split(*restrictedPtr, ",")...)
	}

	namespaceTokenConfigs := "vault_namespaces" + string(os.PathSeparator) + "token_files"
	namespaceRoleConfigs := "vault_namespaces" + string(os.PathSeparator) + "role_files"
	namespacePolicyConfigs := "vault_namespaces" + string(os.PathSeparator) + "policy_files"

	if *namespaceVariable != "" {
		namespaceTokenConfigs = "vault_namespaces" + string(os.PathSeparator) + *namespaceVariable + string(os.PathSeparator) + "token_files"
		namespaceRoleConfigs = "vault_namespaces" + string(os.PathSeparator) + *namespaceVariable + string(os.PathSeparator) + "role_files"
		namespacePolicyConfigs = "vault_namespaces" + string(os.PathSeparator) + *namespaceVariable + string(os.PathSeparator) + "policy_files"
	}

	if *namespaceVariable == "" && !*rotateTokens && !*tokenExpiration && !*updatePolicy && !*updateRole && !*pingPtr {
		if _, err := os.Stat(*seedPtr); os.IsNotExist(err) {
			fmt.Println("Missing required seed folder: " + *seedPtr)
			os.Exit(1)
		}
	}

	if *prodPtr {
		if !strings.HasPrefix(*envPtr, "staging") && !strings.HasPrefix(*envPtr, "prod") {
			flag.Usage()
			os.Exit(1)
		}
	} else {
		if strings.HasPrefix(*envPtr, "staging") || strings.HasPrefix(*envPtr, "prod") {
			flag.Usage()
			os.Exit(1)
		}
	}
	if !*pingPtr && !*newPtr && *tokenPtr == "" {
		utils.CheckWarning("Missing auth tokens", true)
	}

	// If logging production directory does not exist and is selected log to local directory
	if _, err := os.Stat("/var/log/"); *logFilePtr == "/var/log/trcinit.log" && os.IsNotExist(err) {
		*logFilePtr = "./trcinit.log"
	}

	// Initialize logging
	f, err := os.OpenFile(*logFilePtr, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	utils.CheckError(err, true)
	logger := log.New(f, "[INIT]", log.LstdFlags)
	logger.Println("==========Beginning Vault Initialization==========")

	if addrPtr == nil || *addrPtr == "" {
		if *newPtr {
			fmt.Println("Address must be specified using -addr flag")
			os.Exit(1)
		}
		eUtils.AutoAuth(*insecurePtr, nil, nil, tokenPtr, nil, envPtr, addrPtr, *pingPtr, logger)
	}

	// Create a new vault system connection
	v, err := sys.NewVault(*insecurePtr, *addrPtr, *envPtr, *newPtr, *pingPtr, false, logger)
	if err != nil {
		if strings.Contains(err.Error(), "x509: certificate signed by unknown authority") {
			fmt.Printf("Attempting to connect to insecure vault or vault with self signed certificate.  If you really wish to continue, you may add -insecure as on option.\n")
		}
		os.Exit(0)
	}
	if *pingPtr {
		if err != nil {
			fmt.Printf("Ping failure: %v\n", err)
		}
		os.Exit(0)
	}

	utils.LogErrorObject(err, logger, true)

	// Trying to use local, prompt for username/password
	if len(*envPtr) >= 5 && (*envPtr)[:5] == "local" {
		*envPtr, err = utils.LoginToLocal()
		utils.LogErrorObject(err, logger, true)
		logger.Printf("Login successful, using local envronment: %s\n", *envPtr)
	}

	if *devPtr || !*newPtr { // Dev server, initialization taken care of, get root token
		v.SetToken(*tokenPtr)
	} else { // Unseal and grab keys/root token
		totalKeyShard, err := strconv.ParseUint(*keyShardPtr, 10, 32)
		if err != nil {
			fmt.Println("Unable to parse totalKeyShard into int")
		}
		unsealShardPtr, err := strconv.ParseUint(*unsealShardPtr, 10, 32)
		if err != nil {
			fmt.Println("Unable to parse unsealShardPtr into int")
		}
		keyToken, err := v.InitVault(int(totalKeyShard), int(unsealShardPtr))
		utils.LogErrorObject(err, logger, true)
		v.SetToken(keyToken.Token)
		v.SetShards(keyToken.Keys)
		//check error returned by unseal
		_, _, _, err = v.Unseal()
		utils.LogErrorObject(err, logger, true)
	}
	logger.Printf("Succesfully connected to vault at %s\n", *addrPtr)

	if !*newPtr && *namespaceVariable != "" && *namespaceVariable != "vault" && !(*rotateTokens || *updatePolicy || *updateRole || *tokenExpiration) {
		if *initNamespace {
			fmt.Println("Creating tokens, roles, and policies.")
			policyExists, policyErr := il.GetExistsPolicies(namespacePolicyConfigs, v, logger)
			if policyErr != nil {
				utils.LogErrorObject(policyErr, logger, false)
				fmt.Println("Cannot safely determine policy.")
				os.Exit(-1)
			}

			if policyExists {
				fmt.Printf("Policy exists for policy configurations in directory: %s.  Refusing to continue.\n", namespacePolicyConfigs)
				os.Exit(-1)
			}

			roleExists, roleErr := il.GetExistsRoles(namespaceRoleConfigs, v, logger)
			if roleErr != nil {
				utils.LogErrorObject(roleErr, logger, false)
				fmt.Println("Cannot safely determine role.")
			}

			if roleExists {
				fmt.Printf("Role exists for role configurations in directory: %s.  Refusing to continue.\n", namespaceRoleConfigs)
				os.Exit(-1)
			}

			// Special path for generating custom scoped tokens.  This path requires
			// a root token usually.
			//

			// Upload Create/Update new cidr roles.
			fmt.Println("Creating role")
			il.UploadTokenCidrRoles(namespaceRoleConfigs, v, logger)
			// Upload Create/Update policies from the given policy directory

			fmt.Println("Creating policy")
			il.UploadPolicies(namespacePolicyConfigs, v, false, logger)

			// Upload tokens from the given token directory
			fmt.Println("Creating tokens")
			tokens := il.UploadTokens(namespaceTokenConfigs, v, logger)
			if len(tokens) > 0 {
				logger.Println(*namespaceVariable + " tokens successfully created.")
			} else {
				logger.Println(*namespaceVariable + " tokens failed to create.")
			}
			f.Close()
			os.Exit(0)
		} else {
			fmt.Println("initns or rotateTokens required with the namespace paramater.")
			os.Exit(0)
		}

	}

	if !*newPtr && (*updatePolicy || *rotateTokens || *tokenExpiration) {
		if *tokenExpiration {
			fmt.Println("Checking token expiration.")
			roleId, lease, err := v.GetRoleID("bamboo")
			utils.LogErrorObject(err, logger, false)
			fmt.Println("AppRole id: " + roleId + " expiration is set to (zero means never expire): " + lease)
		} else {
			if *rotateTokens {
				fmt.Println("Rotating tokens.")
			}
		}

		if *rotateTokens || *tokenExpiration {
			getOrRevokeError := v.GetOrRevokeTokensInScope(namespaceTokenConfigs, *tokenExpiration, logger)
			if getOrRevokeError != nil {
				fmt.Println("Token revocation or access failure.  Cannot continue.")
				os.Exit(-1)
			}
		}

		if *updateRole {
			// Upload Create/Update new cidr roles.
			fmt.Println("Updating role")
			errTokenCidr := il.UploadTokenCidrRoles(namespaceRoleConfigs, v, logger)
			if errTokenCidr != nil {
				fmt.Println("Role update failed.  Cannot continue.")
				os.Exit(-1)
			} else {
				fmt.Println("Role updated")
			}
		}

		if *updatePolicy {
			// Upload Create/Update policies from the given policy directory
			fmt.Println("Updating policy")
			errTokenPolicy := il.UploadPolicies(namespacePolicyConfigs, v, false, logger)
			if errTokenPolicy != nil {
				fmt.Println("Policy update failed.  Cannot continue.")
				os.Exit(-1)
			} else {
				fmt.Println("Policy updated")
			}
		}

		if *rotateTokens && !*tokenExpiration {
			fmt.Println("Rotating tokens.")

			// Create new tokens.
			tokens := il.UploadTokens(namespaceTokenConfigs, v, logger)
			if !*prodPtr && *namespaceVariable == "vault" {
				//
				// Dev, QA specific token creation.
				//
				tokenMap := map[string]interface{}{}

				mod, err := kv.NewModifier(*insecurePtr, v.GetToken(), *addrPtr, "nonprod", nil, logger) // Connect to vault
				utils.LogErrorObject(err, logger, false)

				mod.Env = "bamboo"

				//Checks if vault is initialized already
				if !mod.Exists("values/metadata") && !mod.Exists("templates/metadata") && !mod.Exists("super-secrets/metadata") {
					fmt.Println("Vault has not been initialized yet")
					os.Exit(1)
				}

				existingTokens, err := mod.ReadData("super-secrets/tokens")
				if err != nil {
					fmt.Println("Read existing tokens failure.  Cannot continue.")
					utils.LogErrorObject(err, logger, false)
					os.Exit(-1)
				}

				// We have names of tokens that were referenced in old role.  Ok to delete the role now.
				//
				// Merge token data.
				//
				// Copy existing.
				for key, valueObj := range existingTokens {
					if value, ok := valueObj.(string); ok {
						tokenMap[key] = value
					} else if stringer, ok := valueObj.(fmt.GoStringer); ok {
						tokenMap[key] = stringer.GoString()
					}
				}

				// Overwrite new.
				for _, token := range tokens {
					// Everything but webapp give access by app role and secret.
					if token.Name != "webapp" {
						tokenMap[token.Name] = token.Value
					}
				}

				//
				// Wipe existing role.
				// Recreate the role.
				//
				resp, role_cleanup := v.DeleteRole("bamboo")
				utils.LogErrorObject(role_cleanup, logger, false)

				if resp.StatusCode == 404 {
					err = v.EnableAppRole()
					utils.LogErrorObject(err, logger, true)
				}

				err = v.CreateNewRole("bamboo", &sys.NewRoleOptions{
					TokenTTL:    "10m",
					TokenMaxTTL: "15m",
					Policies:    []string{"bamboo"},
				})
				utils.LogErrorObject(err, logger, true)

				roleID, _, err := v.GetRoleID("bamboo")
				utils.LogErrorObject(err, logger, true)

				secretID, err := v.GetSecretID("bamboo")
				utils.LogErrorObject(err, logger, true)

				fmt.Printf("Rotated role id and secret id.\n")
				fmt.Printf("Role ID: %s\n", roleID)
				fmt.Printf("Secret ID: %s\n", secretID)

				// Store all new tokens to new appRole.
				warn, err := mod.Write("super-secrets/tokens", tokenMap)
				utils.LogErrorObject(err, logger, true)
				utils.LogWarningsObject(warn, logger, true)
			}
		}
		os.Exit(0)
	}

	// Try to unseal if an old vault and unseal key given
	if !*newPtr && len(*shardPtr) > 0 {
		v.AddShard(*shardPtr)
		prog, t, success, err := v.Unseal()
		utils.LogErrorObject(err, logger, true)
		if !success {
			logger.Printf("Vault unseal progress: %d/%d key shards\n", prog, t)
			logger.Println("============End Initialization Attempt============")
		} else {
			logger.Println("Vault successfully unsealed")
		}
	}

	//TODO: Figure out raft storage initialization for -new flag
	if *newPtr {
		mod, err := kv.NewModifier(*insecurePtr, v.GetToken(), *addrPtr, "nonprod", nil, logger) // Connect to vault
		utils.LogErrorObject(err, logger, true)

		mod.Env = "bamboo"

		if mod.Exists("values/metadata") || mod.Exists("templates/metadata") || mod.Exists("super-secrets/metadata") {
			fmt.Println("Vault has been initialized already...")
			os.Exit(1)
		}

		policyExists, err := il.GetExistsPolicies(namespacePolicyConfigs, v, logger)
		if policyExists || err != nil {
			fmt.Printf("Vault may be initialized already - Policies exists.\n")
			os.Exit(1)
		}

		// Create secret engines
		il.CreateEngines(v, logger)
		// Upload policies from the given policy directory
		il.UploadPolicies(namespacePolicyConfigs, v, false, logger)
		// Upload tokens from the given token directory
		tokens := il.UploadTokens(namespaceTokenConfigs, v, logger)
		if !*prodPtr {
			tokenMap := map[string]interface{}{}
			for _, token := range tokens {
				tokenMap[token.Name] = token.Value
			}

			err = v.EnableAppRole()
			utils.LogErrorObject(err, logger, true)

			err = v.CreateNewRole("bamboo", &sys.NewRoleOptions{
				TokenTTL:    "10m",
				TokenMaxTTL: "15m",
				Policies:    []string{"bamboo"},
			})
			utils.LogErrorObject(err, logger, true)

			roleID, _, err := v.GetRoleID("bamboo")
			utils.LogErrorObject(err, logger, true)

			secretID, err := v.GetSecretID("bamboo")
			utils.LogErrorObject(err, logger, true)

			fmt.Printf("Rotated role id and secret id.\n")
			fmt.Printf("Role ID: %s\n", roleID)
			fmt.Printf("Secret ID: %s\n", secretID)

			// Store all new tokens to new appRole.
			warn, err := mod.Write("super-secrets/tokens", tokenMap)
			utils.LogErrorObject(err, logger, true)
			utils.LogWarningsObject(warn, logger, true)
		}
	}

	// New vaults you can't also seed at same time
	// because you first need tokens to do so.  Only seed if !new.
	if !*newPtr {
		// Seed the vault with given seed directory
		mod, _ := kv.NewModifier(*insecurePtr, *tokenPtr, *addrPtr, *envPtr, nil, logger) // Connect to vault
		mod.Env = *envPtr
		validToken := mod.ValidateEnvironment(mod.Env, true, logger) //This is used to validate token
		if !validToken {
			fmt.Println("Invalid token - token: ", *tokenPtr)
			os.Exit(1)
		}
		var slice = make([]string, 0) //Assign slice with the appriopiate slice
		if len(restrictedSlice) > 0 {
			slice = restrictedSlice
		} else if len(indexSlice) > 0 {
			slice = indexSlice
		}
		il.SeedVault(*insecurePtr, *seedPtr, *addrPtr, v.GetToken(), *envPtr, slice, logger, *servicePtr, *uploadCertPtr)
	}

	logger.SetPrefix("[INIT]")
	logger.Println("=============End Vault Initialization=============")
	logger.Println()

	// Uncomment this when deployed to avoid a hanging root token
	// v.RevokeSelf()
	f.Close()
}
