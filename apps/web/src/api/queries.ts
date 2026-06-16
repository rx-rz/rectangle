import {
	createQueryKeys,
	mergeQueryKeys,
} from "@lukemorales/query-key-factory";

export const authKeys = createQueryKeys("auth", {
	signupWithEmail: null,
	loginWithEmail: null,
	sendOTP: null,
	getGoogleOauthLink: null,
	verifyOTP: null,
});

export const queries = mergeQueryKeys(authKeys);
