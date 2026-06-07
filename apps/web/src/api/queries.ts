import {
	createQueryKeys,
	mergeQueryKeys,
} from "@lukemorales/query-key-factory";

export const authKeys = createQueryKeys("auth", {
	signupWithEmail: null,
	loginWithEmail: null,
	getOTP: null,
	verifyOTP: null,
});

export const queries = mergeQueryKeys(authKeys);
