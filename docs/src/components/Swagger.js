import BrowserOnly from '@docusaurus/core/lib/client/exports/BrowserOnly';
import React from 'react';

const ReadOnlyAuthPlugin = () => ({
    wrapComponents: {
        authorizeBtn: () => () => null,
    },
});

export default function Swagger() {
    return (
        <BrowserOnly>
            {() => {
                const SwaggerUI = require('swagger-ui-react').default;
                require('swagger-ui-react/swagger-ui.css');
                return (
                    <SwaggerUI
                        url="/swagger.yaml"
                        supportedSubmitMethods={[]}
                        plugins={[ReadOnlyAuthPlugin]}
                    />
                );
            }}
        </BrowserOnly>
    );
}
