const psonRE = /^\$pson:[0-9]+$/;

export async function parsePSON(reader) {
	return await new Promise(async (resolve, reject) => {
		const decoder = new TextDecoder();
		let accumulator = '';

		const completers = new Map();
		for (; ;) {
			const { value, done } = await reader.read();
			if (done && accumulator === '') {
				return;
			}

			accumulator += decoder.decode(value);
			for (; ;) {
				const jsonEnd = accumulator.indexOf('\n')
				if (jsonEnd < 0) {
					break;
				}

				const jsonStr = accumulator.slice(0, jsonEnd);
				const obj = swapPromises(JSON.parse(jsonStr), completers);
				resolve(obj);
				complete(obj, completers);
				accumulator = accumulator.slice(jsonEnd + 1);
			}
		}
	})
}

function swapPromises(value, completers) {
	switch (typeof (value)) {
		case 'string':
			if (value.match(psonRE)) {
				const c = new Completer();
				completers.set(value, c);
				return c.promise;
			}
			return value;

		case 'object':
			if (value == null) {
				return value;
			}
			return Object.fromEntries(Object.entries(value)
				.map(([name, value]) => [name, swapPromises(value, completers)]));

		default:
			return value;
	}
}

function complete(obj, completers) {
	if (typeof (obj) !== 'object' || obj == null) {
		return;
	}

	Object.entries(obj)
		.filter(([name, _value]) => name.match(psonRE))
		.forEach(([name, value]) => {
			completers.get(name).resolve(value);
		});
}


class Completer {
	constructor() {
		this.promise = new Promise((resolve, reject) => {
			this.resolve = resolve
			this.reject = reject
		})
	}
}
