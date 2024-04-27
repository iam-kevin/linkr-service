const { randomBytes } = require("crypto");
const sqlite = require("@libsql/client");
const cuid = require("@paralleldrive/cuid2");

// connection to the database to seed initiale users
const client = sqlite.createClient({
	url: process.env.DATABASE_URL,
	authToken: process.env.AUTH_TOKEN,
});

const SUPPORTED_ROLES = ["admin", "read-wrte", "read-only", "write-only"];

const DELIMITER = ":";

/**
 * @typedef {'admin' | 'read-write' | 'read-only' | 'write-only'} RoleType
 */

/**
 * decodes input of share `username:role` or `username` into
 * the tuple [username, role|defaultrole]
 *
 * @param {string} input
 * @returns {[string, RoleType]}
 */
function decodetext(input) {
	const vals = input.split(DELIMITER);
	if (vals.length >= 3 || vals.length == 0) {
		throw new Error("invalid input. must be of share username[:role]");
	}

	const [username, role] = vals;

	if (role != undefined) {
		if (!SUPPORTED_ROLES.includes(role)) {
			throw new Error(
				`unknown role type '${role}'. supported roles are ${SUPPORTED_ROLES}`
			);
		}
	}

	return [username, role ?? "read-write"];
}

/**
 * seed the users
 *
 * @param {sqlite.Client} sqlite
 * @param {Array<[string, RoleType]>} userroles
 */
async function seed(sqlite, userroles) {
	const created = [];

	await Promise.all(
		userroles.map(([user, role]) => {
			let id = cuid.createId();

			const d = new Date();
			const month = `${d.getMonth() + 1}`.padStart(2, "0");
			const date = `${d.getDate() + 1}`.padStart(2, "0");
			id = `api_${id}-${d.getFullYear()}${month}${date}`;

			// create
			const key = Buffer.from(randomBytes(32)).toString("base64");

			created.push({
				id,
				signingKey: key,
				user: user,
				role,
			});

			return sqlite.execute({
				sql: `INSERT INTO "ApiClient" (id, username, description, scope, signing_key, created_at, updated_at) VALUES (?,?,?,?,?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
				args: [id, user, null, role, key],
			});
		})
	);

	console.log("Keys created", created);
	process.exit(1);
}

function main() {
	const tuples_userroles = process.argv.slice(2, process.argv.length);

	seed(client, tuples_userroles.map(decodetext));
}

main();
