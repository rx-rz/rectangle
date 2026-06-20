export const API_ROUTES = {
	me: "/me",
	auth: {
		signup: "/auth/signup/email",
		login: "/auth/login/email",
		oauth: {
			github: "/auth/github/start",
			google: "/auth/google/start",
		},
		otp: {
			send: "/auth/otp/send",
			verify: "/auth/otp/verify",
		},
	},
} as const;
