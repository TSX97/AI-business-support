interface AuthMessagesProps {
    error: string;
    success: string;
}

function AuthMessages({ error, success }: AuthMessagesProps) {
    return (
        <>
            {error && (
                <p className="message error" role="alert">
                    {error}
                </p>
            )}

            {success && (
                <p className="message success" role="status">
                    {success}
                </p>
            )}
        </>
    );
}

export default AuthMessages;
