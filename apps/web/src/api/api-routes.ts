export const API_ROUTES = {
	auth: {
		signup: "/auth/signup/email",
		login: "/auth/login/email",
		otp: {
			send: "/auth/otp/send",
			verify: "/auth/otp/verify",
		},
	},
} as const;
