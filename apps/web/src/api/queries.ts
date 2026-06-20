import {
	createQueryKeys,
	mergeQueryKeys,
} from "@lukemorales/query-key-factory";

export const authKeys = createQueryKeys("auth", {
	me: null,
	signupWithEmail: null,
	loginWithEmail: null,
	sendOTP: null,
	getGithubOauthLink: null,
	getGoogleOauthLink: null,
	verifyOTP: null,
});

export const queries = mergeQueryKeys(authKeys);
