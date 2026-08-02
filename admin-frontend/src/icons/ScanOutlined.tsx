import Icon from '@ant-design/icons';
import type { GetProps } from 'antd';
import React from 'react';

type CustomIconComponentProps = GetProps<typeof Icon>;

const SVG: React.FC = () => (
	<svg
		width="1em"
		height="1em"
		viewBox="0 0 1024 1024"
		fill="none"
		xmlns="http://www.w3.org/2000/svg"
	>
		<style>
			{`
				.scan-outlined-line {
					animation: scan-outlined-sweep 2.2s ease-in-out infinite;
					transform-origin: center;
				}

				.scan-outlined-glow {
					animation: scan-outlined-pulse 2.2s ease-in-out infinite;
					transform-origin: center;
				}

				@keyframes scan-outlined-sweep {
					0%,
					100% {
						transform: translateY(-120px);
						opacity: 0;
					}

					15% {
						opacity: 0.95;
					}

					50% {
						transform: translateY(120px);
						opacity: 1;
					}

					85% {
						opacity: 0.95;
					}
				}

				@keyframes scan-outlined-pulse {
					0%,
					100% {
						opacity: 0.18;
					}

					50% {
						opacity: 0.36;
					}
				}
			`}
		</style>

		<g
			stroke="currentColor"
			strokeLinecap="round"
			strokeLinejoin="round"
		>
			<path
				d="M188 328V236c0-26.51 21.49-48 48-48h92M696 188h92c26.51 0 48 21.49 48 48v92M836 696v92c0 26.51-21.49 48-48 48h-92M328 836h-92c-26.51 0-48-21.49-48-48v-92"
				strokeWidth="54"
			/>

			<rect
				x="264"
				y="264"
				width="496"
				height="496"
				rx="88"
				fill="currentColor"
				opacity="0.08"
				strokeWidth="40"
			/>

			<g
				className="scan-outlined-glow"
				opacity="0.2"
			>
				<path
					d="M346 226c50-34 111-54 178-54 89 0 167 35 224 93"
					strokeWidth="34"
				/>
				<path
					d="M798 474c0 92-37 175-97 235"
					strokeWidth="34"
				/>
			</g>

			<g strokeWidth="34">
				<rect
					x="340"
					y="340"
					width="122"
					height="122"
					rx="26"
				/>
				<rect
					x="372"
					y="372"
					width="58"
					height="58"
					rx="12"
					fill="currentColor"
					stroke="none"
				/>

				<rect
					x="562"
					y="340"
					width="122"
					height="122"
					rx="26"
				/>
				<rect
					x="594"
					y="372"
					width="58"
					height="58"
					rx="12"
					fill="currentColor"
					stroke="none"
				/>

				<rect
					x="340"
					y="562"
					width="122"
					height="122"
					rx="26"
				/>
				<rect
					x="372"
					y="594"
					width="58"
					height="58"
					rx="12"
					fill="currentColor"
					stroke="none"
				/>
			</g>

			<g
				fill="currentColor"
				stroke="none"
			>
				<rect
					x="546"
					y="546"
					width="48"
					height="48"
					rx="10"
				/>
				<rect
					x="610"
					y="546"
					width="48"
					height="48"
					rx="10"
				/>
				<rect
					x="674"
					y="546"
					width="48"
					height="48"
					rx="10"
				/>
				<rect
					x="546"
					y="610"
					width="48"
					height="48"
					rx="10"
				/>
				<rect
					x="642"
					y="610"
					width="80"
					height="48"
					rx="10"
				/>
				<rect
					x="578"
					y="674"
					width="48"
					height="48"
					rx="10"
				/>
				<rect
					x="674"
					y="674"
					width="48"
					height="48"
					rx="10"
				/>
			</g>

			<g className="scan-outlined-line">
				<path
					d="M320 512H704"
					strokeWidth="28"
					opacity="0.95"
				/>
				<path
					d="M344 512H680"
					strokeWidth="74"
					opacity="0.12"
				/>
			</g>
		</g>
	</svg>
);

const ScanOutlined: React.FC<Partial<CustomIconComponentProps>> = props => (
	<Icon
		component={SVG}
		{...props}
	/>
);

export default ScanOutlined;
